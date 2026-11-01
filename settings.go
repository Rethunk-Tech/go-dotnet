package dotnet

import (
	"encoding/binary"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Setting is one BepInEx config entry a plugin binds with a constant section, key and default.
type Setting struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	// Type is "bool", "int", "uint", "long", "ulong", "short", "ushort", "byte", "sbyte", "float", "double", "string"
	// or "enum".
	Type string `json:"type"`
	// Default is written as BepInEx writes it to the .cfg; empty when the reader cannot tell (an enum it cannot name).
	Default     string `json:"default"`
	Description string `json:"description,omitempty"`
	// Min and Max are an AcceptableValueRange's bounds, in the same form as Default.
	Min string `json:"min,omitempty"`
	Max string `json:"max,omitempty"`
	// Values are an enum's names, in declaration order.
	Values []string `json:"values,omitempty"`
	// Sync is "host-enforced" or "host-recommended" when the entry is shared with the lobby by Mortar's ConfigSync or
	// by CSync straight from the Bind call, else empty.
	Sync string `json:"sync,omitempty"`
}

// Settings lists the config entries the assembly at path binds through BepInEx's ConfigFile.Bind (or CSync's
// BindSyncedEntry) where the section, key and default are constants in the call, in IL order, each section and key
// once. A Bind whose default is computed is left out, as is one through a helper that takes the section or key as a
// parameter.
func Settings(path string) ([]Setting, error) {
	return withFile(path, func(m *assembly) ([]Setting, error) {
		u := m.usageNames()
		r := settingReader{m: m, u: u, seen: map[string]bool{}, out: []Setting{}}
		for md := 1; md <= m.rows[tMethodDef]; md++ {
			if rva := m.cell(tMethodDef, md, 0); rva != 0 {
				if code := ilCode(m.at(int(rva))); code != nil {
					r.body(code)
				}
			}
		}
		return r.out, nil
	})
}

const (
	opLdstr    = 0x72
	opLdnull   = 0x14
	opLdcI4M1  = 0x15
	opLdcI48   = 0x1E
	opLdcI4S   = 0x1F
	opLdcI4    = 0x20
	opLdcI8    = 0x21
	opLdcR4    = 0x22
	opLdcR8    = 0x23
	opCall     = 0x28
	opCallvirt = 0x6F
	opNewobj   = 0x73
	opConvU8   = 0x6E

	bindName   = "BepInEx.Configuration.ConfigFile::Bind"
	csyncBind  = "::BindSyncedEntry"
	shareName  = "ConfigSync.SyncedConfig::Share"
	rangeCtor  = "BepInEx.Configuration.AcceptableValueRange`1::.ctor"
	enumBase   = "System.Enum"
	enumValues = "value__"
)

// constant is a value an instruction pushes: a string, an integer or a floating-point number.
type constant struct {
	kind byte // 's', 'i' or 'f'
	s    string
	i    int64
	f    float64
	r4   bool
}

// step is one instruction in a body, with the constant it pushes if it pushes one. Conversions are dropped, so a
// long default written as ldc.i4 then conv.i8 reads as the constant.
type step struct {
	in instr
	c  *constant
}

type settingReader struct {
	m    *assembly
	u    usageNames
	seen map[string]bool
	out  []Setting
}

// pendingBind is a Bind call under way: its section, key and default, and what follows them before the call.
type pendingBind struct {
	section, key string
	def          constant
	after        int
	desc         *string
	min, max     *constant
}

func (r *settingReader) body(code []byte) {
	var steps []step
	for in := range ops(code) {
		if (in.op >= 0x67 && in.op <= 0x6E) || in.op == 0x76 {
			// conv.u8 widens without sign: a long above int's range is written as its low 32 bits and this.
			if last := len(steps) - 1; in.op == opConvU8 && last >= 0 && steps[last].c != nil && steps[last].c.kind == 'i' {
				steps[last].c.i = unsigned32(steps[last].c.i)
			}
			continue
		}
		steps = append(steps, step{in: in, c: r.constant(code, in)})
	}
	var cur *pendingBind
	for i, st := range steps {
		if (cur == nil || i >= cur.after) && i+2 < len(steps) && isString(steps[i].c) && isString(steps[i+1].c) && steps[i+2].c != nil {
			cur = &pendingBind{section: steps[i].c.s, key: steps[i+1].c.s, def: *steps[i+2].c, after: i + 3}
			continue
		}
		if cur == nil || i < cur.after {
			continue
		}
		switch {
		case isString(st.c) && cur.desc == nil:
			cur.desc = &st.c.s
		case st.in.op == opNewobj && r.u.name(r.m, st.in.arg) == rangeCtor && i >= 2 && steps[i-2].c != nil && steps[i-1].c != nil:
			cur.min, cur.max = steps[i-2].c, steps[i-1].c
		case st.in.op == opCall || st.in.op == opCallvirt:
			name := r.u.name(r.m, st.in.arg)
			csync := strings.HasSuffix(name, csyncBind)
			if name != bindName && !csync {
				continue
			}
			sync := ""
			if csync {
				sync = "host-enforced"
			} else if i+2 < len(steps) && steps[i+1].c != nil && steps[i+1].c.kind == 'i' && (steps[i+2].in.op == opCall || steps[i+2].in.op == opCallvirt) && r.u.name(r.m, steps[i+2].in.arg) == shareName {
				// ConfigSync's Sync enum: HostRecommended is 0, HostEnforced 1.
				sync = "host-recommended"
				if steps[i+1].c.i == 1 {
					sync = "host-enforced"
				}
			}
			r.emit(cur, st.in.arg, sync)
			cur = nil
		}
	}
}

func isString(c *constant) bool { return c != nil && c.kind == 's' }

// constant reads the value an instruction pushes, or nil when it pushes none.
func (r *settingReader) constant(code []byte, in instr) *constant {
	switch {
	case in.op == opLdstr:
		return &constant{kind: 's', s: r.m.userString(in.arg)}
	case in.op >= opLdcI4M1 && in.op <= opLdcI48:
		return &constant{kind: 'i', i: int64(in.op - opLdcI4M1 - 1)}
	case in.op == opLdcI4S:
		return &constant{kind: 'i', i: twos(int64(in.arg), 8)}
	case in.op == opLdcI4:
		return &constant{kind: 'i', i: twos(int64(in.arg), 32)}
	case in.op == opLdcI8:
		return &constant{kind: 'i', i: twos(int64(le32(code[in.at+4:])), 32)<<32 | int64(le32(code[in.at:]))}
	case in.op == opLdcR4:
		return &constant{kind: 'f', f: float64(math.Float32frombits(in.arg)), r4: true}
	case in.op == opLdcR8:
		return &constant{kind: 'f', f: math.Float64frombits(binary.LittleEndian.Uint64(code[in.at:]))}
	}
	return nil
}

// twos reads the low bits of v as a two's-complement number.
func twos(v int64, bits uint) int64 {
	if v >= 1<<(bits-1) {
		return v - 1<<bits
	}
	return v
}

// unsigned32 reads an int32 as the uint32 with the same bits.
func unsigned32(v int64) int64 {
	if v < 0 {
		return v + 1<<32
	}
	return v
}

// userString reads a #US heap entry: UTF-16 code units after a compressed length, with one trailing flag byte.
func (m *assembly) userString(token uint32) string {
	i := int(token & 0xFFFFFF)
	n, size := compressed(m.us[i:])
	b := m.us[i+size : i+size+int(n)]
	if len(b) > 0 {
		b = b[:len(b)-1]
	}
	units := make([]uint16, len(b)/2)
	for k := range units {
		units[k] = binary.LittleEndian.Uint16(b[2*k:])
	}
	return string(utf16.Decode(units))
}

func (r *settingReader) emit(cur *pendingBind, call uint32, sync string) {
	id := cur.section + "\n" + cur.key
	if r.seen[id] {
		return
	}
	typ, values, names := r.bindType(call)
	if typ == "" {
		return
	}
	s := Setting{Section: cur.section, Key: cur.key, Type: typ, Default: format(typ, cur.def, names), Values: values, Sync: sync}
	if cur.desc != nil {
		s.Description = *cur.desc
	}
	if cur.min != nil {
		s.Min, s.Max = format(typ, *cur.min, names), format(typ, *cur.max, names)
	}
	r.seen[id] = true
	r.out = append(r.out, s)
}

var elementTypes = map[byte]string{
	0x02: "bool", 0x04: "sbyte", 0x05: "byte", 0x06: "short", 0x07: "ushort", 0x08: "int", 0x09: "uint",
	0x0A: "long", 0x0B: "ulong", 0x0C: "float", 0x0D: "double", 0x0E: "string",
}

// bindType is the type argument of a generic Bind call; for an enum the module defines, also its names in order and
// by value. Empty for a call that names no instantiation or a type the reader does not handle.
func (r *settingReader) bindType(call uint32) (typ string, values []string, names map[int64]string) {
	if call>>24 != tMethodSpec {
		return "", nil, nil
	}
	inst := r.m.blob(r.m.cell(tMethodSpec, int(call&0xFFFFFF), 1))
	if len(inst) < 3 || inst[0] != 0x0A {
		return "", nil, nil
	}
	sig := inst[2:]
	if t, ok := elementTypes[sig[0]]; ok {
		return t, nil, nil
	}
	if sig[0] != 0x11 || len(sig) < 2 {
		return "", nil, nil
	}
	v, _ := compressed(sig[1:])
	table, row := decode(cTypeDefOrRef, v)
	if table == tTypeDef {
		if values, names, ok := r.m.enumOf(row); ok {
			return "enum", values, names
		}
		return "", nil, nil
	}
	// An enum from another assembly: its names are not here, so the default stays unknown.
	return "enum", nil, nil
}

// enumOf reads a module's enum: its member names in order and by value; ok is false when the type is no enum.
func (m *assembly) enumOf(t int) (values []string, names map[int64]string, ok bool) {
	table, row := decode(cTypeDefOrRef, m.cell(tTypeDef, t, 3))
	ns, name := m.typeOf(table, row)
	if ns+"."+name != enumBase {
		return nil, nil, false
	}
	constants := map[int][]byte{}
	for c := 1; c <= m.rows[tConstant]; c++ {
		if table, row := decode(cHasConstant, m.cell(tConstant, c, 1)); table == tField {
			constants[row] = m.blob(m.cell(tConstant, c, 2))
		}
	}
	names = map[int64]string{}
	for f := range m.fieldsOf(t) {
		field := m.str(m.cell(tField, f, 1))
		value, has := constants[f]
		if field == enumValues || !has {
			continue
		}
		var n int64
		for _, b := range slices.Backward(value) {
			n = n<<8 | int64(b)
		}
		if len(value) > 0 && len(value) < 8 {
			n = twos(n, uint(8*len(value)))
		}
		values = append(values, field)
		if _, dup := names[n]; !dup {
			names[n] = field
		}
	}
	return values, names, true
}

// format writes a constant as BepInEx's TOML converter writes a value of typ: invariant-culture numbers, lowercase
// booleans, an enum by its name.
func format(typ string, c constant, names map[int64]string) string {
	switch typ {
	case "string":
		if c.kind == 's' {
			return c.s
		}
	case "bool":
		if c.kind == 'i' {
			return strconv.FormatBool(c.i != 0)
		}
	case "enum":
		if c.kind == 'i' {
			return names[c.i]
		}
	case "float", "double":
		bits := 64
		if typ == "float" {
			bits = 32
		}
		f := c.f
		if c.kind == 'i' {
			f = float64(c.i)
		}
		return strings.ToUpper(strconv.FormatFloat(f, 'g', -1, bits))
	case "uint":
		if c.kind == 'i' {
			return strconv.FormatInt(unsigned32(c.i), 10)
		}
	case "ulong":
		if c.kind == 'i' {
			if c.i >= 0 {
				return strconv.FormatInt(c.i, 10)
			}
			return new(big.Int).Add(big.NewInt(c.i), new(big.Int).Lsh(big.NewInt(1), 64)).String()
		}
	default:
		if c.kind == 'i' {
			return strconv.FormatInt(c.i, 10)
		}
	}
	return ""
}
