package dotnet

import (
	"slices"
	"testing"
)

func TestSignatureTypesMatchHowCSharpWritesThem(t *testing.T) {
	n := sigNames{m: &assembly{}}
	for _, c := range []struct {
		sig  []byte
		want string
	}{
		{nil, "object"},
		{[]byte{0x00}, "object"},
		// 0x45 pinned and 0x1F required-modifier: C# drops both from the type name.
		{[]byte{0x45, 0x1F, 0x01, 0x08}, "int"},
		{[]byte{0x0F, 0x1D, 0x08}, "int[]*"},
		{[]byte{0x10, 0x08}, "ref int"},
		// rank, then sizes and lower bounds that the spelling ignores
		{[]byte{0x14, 0x08, 0x02, 0x01, 0x04, 0x00}, "int[,]"},
		{[]byte{0x1B}, "IntPtr"},
		{[]byte{0x13, 0x02}, "T2"},
		{[]byte{0x1E, 0x01}, "TM1"},
	} {
		if got, _ := n.csType(c.sig, 0); got != c.want {
			t.Errorf("%x: %q, want %q", c.sig, got, c.want)
		}
	}
	returns, params := n.methodTypes([]byte{0x10, 0x01, 0x01, 0x08, 0x0E})
	if returns != "int" || !slices.Equal(params, []string{"string"}) {
		t.Fatalf("generic method = %q %v", returns, params)
	}
}

func TestSignatureNamesFollowTheModuleTables(t *testing.T) {
	im := newImage()
	im.table(tTypeSpec, 1, 1)
	im.set(tTypeSpec, 1, 0, 1)
	im.table(tGenParam, 2, 4)
	im.set(tGenParam, 1, 0, 0)
	im.set(tGenParam, 1, 2, 1<<1) // TypeDef row 1
	im.set(tGenParam, 1, 3, im.str("TItem"))
	im.set(tGenParam, 2, 0, 0)
	im.set(tGenParam, 2, 2, 1<<1|1) // MethodDef row 1
	im.set(tGenParam, 2, 3, im.str("TResult"))
	m := im.done()
	spec := []byte{0x1D, 0x08}
	m.blobs = append([]byte{0, byte(len(spec) & 0xFF)}, spec...)
	n := sigNames{m: m, typ: 1, method: 1}
	if got, _ := n.csType([]byte{0x12, 0x06}, 0); got != "int[]" {
		t.Fatalf("TypeSpec = %q", got)
	}
	if got, _ := n.csType([]byte{0x13, 0x00}, 0); got != "TItem" {
		t.Fatalf("type param = %q", got)
	}
	if got, _ := n.csType([]byte{0x1E, 0x00}, 0); got != "TResult" {
		t.Fatalf("method param = %q", got)
	}
}

func TestASignaturePastItsBoundIsRefused(t *testing.T) {
	mustPanicTooDeep(t, "csType depth", func() {
		sigNames{m: &assembly{}}.csType([]byte{0x08}, maxDepth+1)
	})
}
