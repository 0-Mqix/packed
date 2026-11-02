package packed

import (
	"testing"
)

func device(fields ...LayoutField) *Layout {
	return &Layout{Kind: LayoutStruct, Type: "Device", Name: "Device", Function: "Device", Fields: fields}
}

func field(name string, kind LayoutKind) LayoutField {
	return LayoutField{Name: name, Layout: &Layout{Kind: kind, Type: kind.String()}}
}

func TestLayoutDescription(t *testing.T) {
	user := &Layout{Kind: LayoutStruct, Type: "User", Name: "User", Function: "User", Fields: []LayoutField{field("Name", LayoutString)}}
	root := device(
		field("ID", LayoutInt),
		LayoutField{Name: "Owner", Layout: &Layout{Kind: LayoutPointer, Type: "*User", Element: user}},
		LayoutField{Name: "Status", Layout: &Layout{Kind: LayoutUint8, Type: "Status", Name: "Status"}},
		LayoutField{Name: "Scores", Layout: &Layout{Kind: LayoutMap, Type: "map[string][2]float64",
			Key:     &Layout{Kind: LayoutString, Type: "string"},
			Element: &Layout{Kind: LayoutArray, Type: "[2]float64", Length: 2, Element: &Layout{Kind: LayoutFloat64, Type: "float64"}}}},
	)
	root.Fields = append(root.Fields, LayoutField{Name: "Children", Layout: &Layout{Kind: LayoutSlice, Type: "[]Device", Element: root}})

	want := "packed dynamic 1\n#0 Device{ID int; Owner *#1 User{Name string}; Status Status(uint8); Scores map[string][2]float64; Children []#0}"
	if got := root.Description(); got != want {
		t.Fatalf("description\n got %s\nwant %s", got, want)
	}
}

// TestSchemaChanges: a schema changes with every change to a type that
// changes its layout or its names.
func TestSchemaChanges(t *testing.T) {
	base := device(field("ID", LayoutInt64), field("Name", LayoutString)).Schema()

	for name, changed := range map[string]*Layout{
		"renamed field":   device(field("Identifier", LayoutInt64), field("Name", LayoutString)),
		"reordered":       device(field("Name", LayoutString), field("ID", LayoutInt64)),
		"another width":   device(field("ID", LayoutInt32), field("Name", LayoutString)),
		"another field":   device(field("ID", LayoutInt64), field("Name", LayoutString), field("Flag", LayoutBool)),
		"a field removed": device(field("ID", LayoutInt64)),
		"renamed type":    {Kind: LayoutStruct, Name: "Machine", Fields: []LayoutField{field("ID", LayoutInt64), field("Name", LayoutString)}},
	} {
		if changed.Schema() == base {
			t.Errorf("%s: the schema stayed %016x", name, base)
		}
	}

	if again := device(field("ID", LayoutInt64), field("Name", LayoutString)).Schema(); again != base {
		t.Error("one layout has two schemas")
	}
}

func TestLayoutMinimumSize(t *testing.T) {
	layout := device(
		field("Active", LayoutBool),
		LayoutField{Name: "Owner", Layout: &Layout{Kind: LayoutPointer, Element: device()}},
		field("Visible", LayoutBool), // the three share a byte
		field("ID", LayoutUint16),
		field("Name", LayoutString),
		LayoutField{Name: "Codes", Layout: &Layout{Kind: LayoutArray, Length: 3, Element: &Layout{Kind: LayoutString}}},
		field("Created", LayoutTime),
		field("Done", LayoutBool),
	)
	if got, want := layout.MinimumSize(), 1+2+4+3*4+8+1; got != want {
		t.Fatalf("minimum size %d, want %d", got, want)
	}
}
