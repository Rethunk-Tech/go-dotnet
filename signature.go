package dotnet

import (
	"strconv"
	"strings"
)

// csNames are the element types C# writes with a keyword.
var csNames = map[byte]string{
	0x01: "void", 0x02: "bool", 0x03: "char", 0x04: "sbyte", 0x05: "byte", 0x06: "short", 0x07: "ushort", 0x08: "int",
	0x09: "uint", 0x0A: "long", 0x0B: "ulong", 0x0C: "float", 0x0D: "double", 0x0E: "string", 0x16: "System.TypedReference",
	0x18: "System.IntPtr", 0x19: "System.UIntPtr", 0x1C: "object",
}

// sigNames reads signature types as C# writes them, in the context of the method (or field) whose signature it is: a
// type parameter is named by the type or method that declares it.
type sigNames struct {
	m      *assembly
	outer  map[int]int
	typ    int
	method int
}

// csType names the signature type at the start of sig ("int", "GameNetcodeStuff.PlayerControllerB",
// "System.Collections.Generic.List<int>", "Tile[]", "ref int", "T") and returns the signature after it. A type from the
// System namespace drops it, as a file that uses System writes it; a nested type joins its outer type with a dot.
func (n sigNames) csType(sig []byte, depth int) (string, []byte) {
	if depth > maxDepth {
		panic(tooDeep{})
	}
	if len(sig) == 0 {
		return "object", nil
	}
	for len(sig) > 1 && (sig[0] == 0x1F || sig[0] == 0x20 || sig[0] == 0x45) {
		if sig[0] == 0x45 {
			sig = sig[1:]
			continue
		}
		_, k := compressed(sig[1:])
		sig = sig[1+k:]
	}
	el, rest := sig[0], sig[1:]
	if name, ok := csNames[el]; ok {
		return strings.TrimPrefix(name, "System."), rest
	}
	switch el {
	case 0x0F:
		inner, after := n.csType(rest, depth+1)
		return inner + "*", after
	case 0x10:
		inner, after := n.csType(rest, depth+1)
		return "ref " + inner, after
	case 0x11, 0x12:
		v, k := compressed(rest)
		return n.typeName(v, depth), rest[k:]
	case 0x13, 0x1E:
		v, k := compressed(rest)
		return n.typeParam(el == 0x1E, int(v)), rest[k:]
	case 0x15:
		v, k := compressed(rest[1:])
		base := strings.TrimSuffix(n.typeName(v, depth), "`")
		if i := strings.LastIndexByte(base, '`'); i >= 0 {
			base = base[:i]
		}
		count, c := compressed(rest[1+k:])
		after := rest[1+k+c:]
		args := make([]string, count)
		for i := range args {
			args[i], after = n.csType(after, depth+1)
		}
		return base + "<" + strings.Join(args, ", ") + ">", after
	case 0x1D:
		inner, after := n.csType(rest, depth+1)
		return inner + "[]", after
	case 0x14:
		inner, after := n.csType(rest, depth+1)
		rank, k := compressed(after)
		after = after[k:]
		for range 2 {
			count, c := compressed(after)
			after = after[c:]
			for range count {
				_, s := compressed(after)
				after = after[s:]
			}
		}
		return inner + "[" + strings.Repeat(",", max(int(rank)-1, 0)) + "]", after
	case 0x1B:
		return "IntPtr", nil
	}
	return "object", nil
}

// typeName names a TypeDefOrRef-coded type: the module's own with its namespace, a referenced one by its scope.
func (n sigNames) typeName(v uint32, depth int) string {
	table, row := decode(cTypeDefOrRef, v)
	var full string
	switch table {
	case tTypeDef:
		full = n.m.nestedName(row, n.outer)
	case tTypeRef:
		ns, name := n.m.typeRefAt(row, depth+1)
		full = name
		if ns != "" && ns != "System" {
			full = ns + "." + name
		}
	case tTypeSpec:
		name, _ := n.csType(n.m.blob(n.m.cell(tTypeSpec, row, 0)), depth+1)
		return name
	}
	return strings.ReplaceAll(full, "+", ".")
}

// typeParam names the index'th type parameter of the method (method true) or type whose signature this is.
func (n sigNames) typeParam(method bool, index int) string {
	table, owner := tTypeDef, n.typ
	if method {
		table, owner = tMethodDef, n.method
	}
	for row := 1; row <= n.m.rows[tGenParam]; row++ {
		t, o := decode(cTypeOrMethodDef, n.m.cell(tGenParam, row, 2))
		if t == table && o == owner && int(n.m.cell(tGenParam, row, 0)) == index {
			return n.m.str(n.m.cell(tGenParam, row, 3))
		}
	}
	if method {
		return "TM" + strconv.Itoa(index)
	}
	return "T" + strconv.Itoa(index)
}

// methodTypes is a method signature's return type and parameter types as C# writes them.
func (n sigNames) methodTypes(sig []byte) (returns string, params []string) {
	p := 1
	if sig[0]&0x10 != 0 {
		_, k := compressed(sig[p:])
		p += k
	}
	count, k := compressed(sig[p:])
	rest := sig[p+k:]
	returns, rest = n.csType(rest, 0)
	params = make([]string, count)
	for i := range params {
		params[i], rest = n.csType(rest, 0)
	}
	return returns, params
}
