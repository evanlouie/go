package yaml

import (
	"reflect"
	"testing"
)

func TestDecodeMaps(t *testing.T) {
	type args struct {
		document []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "empty",
			args:    args{},
			want:    nil,
			wantErr: false,
		},
		{
			name: "single empty map",
			args: args{
				document: []byte(`---`),
			},
			want:    []map[string]interface{}{nil},
			wantErr: false,
		},
		{
			name: "single non-empty map",
			args: args{
				document: []byte(`---
foo:
  bar: 123`),
			},
			want: []map[string]interface{}{
				{"foo": map[string]interface{}{"bar": 123}},
			},
			wantErr: false,
		},
		{
			name: "multi map doc",
			args: args{
				document: []byte(`---
foo:
  bar: 123
---
baz:
  another:
    nested: map
  aList:
    - foo
    - bar`),
			},
			want: []map[string]interface{}{
				{"foo": map[string]interface{}{
					"bar": 123,
				}},
				{"baz": map[string]interface{}{
					"another": map[string]interface{}{
						"nested": "map",
					},
					"aList": []interface{}{"foo", "bar"},
				}},
			},
			wantErr: false,
		},
		{
			name: "single non-map doc",
			args: args{
				document: []byte("foobar"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "multiple with one non-map",
			args: args{
				document: []byte(`---
foo: bar
---
baz: zoo
---
- I
- Am
- Not
- A
- Map`),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeMaps(tt.args.document)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeMaps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DecodeMaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	type args struct {
		doc []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		wantErr bool
	}{
		{
			"empty",
			args{},
			nil,
			false,
		},
		{
			"single doc",
			args{[]byte(`foobar`)},
			[]interface{}{`foobar`},
			false,
		},
		{
			"multi doc only one document header",
			args{[]byte(`foobar
---
123`)},
			[]interface{}{"foobar", 123},
			false,
		},
		{
			"multi doc ",
			args{[]byte(`---
foobar
---
123
---
foo:
  bar: 123
---
- 1
- foo
- bar
- true`)},
			[]interface{}{
				`foobar`,
				123,
				map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": 123,
					},
				},
				[]interface{}{
					1,
					"foo",
					"bar",
					true,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.args.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() = %v, want %v", got, tt.want)
			}
		})
	}
}
