package main

import (
	"bytes"
	"fmt"
	h "html/template"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	eproto "github.com/pinpt/protoc-gen-es6rpc/proto"
)

func strp(v string) *string {
	return &v
}

func extractComments(file *descriptor.FileDescriptorProto) map[string]*descriptor.SourceCodeInfo_Location {
	comments := make(map[string]*descriptor.SourceCodeInfo_Location)
	for _, loc := range file.GetSourceCodeInfo().GetLocation() {
		if loc.LeadingComments == nil {
			continue
		}
		var p []string
		for _, n := range loc.Path {
			p = append(p, strconv.Itoa(int(n)))
		}
		comments[strings.Join(p, ",")] = loc
	}
	return comments
}

func extractTypes(types []*descriptor.DescriptorProto) map[string]*descriptor.DescriptorProto {
	typeMap := make(map[string]*descriptor.DescriptorProto)
	for _, t := range types {
		if t.GetName() != "" {
			typeMap[t.GetName()] = t
		}
	}
	return typeMap
}

func getServicePath(serviceIndex int) string {
	return fmt.Sprintf("6,%d", serviceIndex)
}

func getMessagePath(messageIndex int) string {
	return fmt.Sprintf("4,%d", messageIndex)
}

func getFieldPath(messageIndex int, fieldIndex int) string {
	return fmt.Sprintf("4,%d,2,%d", messageIndex, fieldIndex)
}

func getServiceMethodPath(serviceIndex int, methodIndex int) string {
	return fmt.Sprintf("6,%d,2,%d", serviceIndex, methodIndex)
}

func messageComment(index int, comments map[string]*descriptor.SourceCodeInfo_Location, indent string) string {
	k := getMessagePath(index)
	c := comments[k]
	return strings.Replace(strings.TrimSpace(c.GetLeadingComments()), "\n", "\n"+indent+"//", -1)
}

func methodComment(service int, method int, comments map[string]*descriptor.SourceCodeInfo_Location, indent string) string {
	k := getServiceMethodPath(service, method)
	c := comments[k]
	comment := strings.TrimSpace(c.GetLeadingComments())
	return strings.Replace(comment, "\n", "\n"+indent+"//", -1)
}

func getValueExtension(field *descriptor.FieldDescriptorProto, ed *proto.ExtensionDesc) string {
	// log.Println(util.Stringify(field, true))
	if field.Options != nil {
		e, _ := proto.GetExtension(field.GetOptions(), ed)
		if s, ok := e.(*eproto.FieldOptions); ok {
			if s.Value != "" {
				return s.Value
			}
		}
	}
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE, descriptor.FieldDescriptorProto_TYPE_FLOAT,
		descriptor.FieldDescriptorProto_TYPE_INT64, descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_FIXED32, descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32, descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_SINT32, descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_ENUM:
		{
			return "0"
		}
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		{
			return "false"
		}
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		{
			return "null"
		}
	}
	return "undefined"
}

func getJSType(field *descriptor.FieldDescriptorProto) string {
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE, descriptor.FieldDescriptorProto_TYPE_FLOAT,
		descriptor.FieldDescriptorProto_TYPE_INT64, descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_FIXED32, descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32, descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_SINT32, descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_ENUM:
		{
			return "number"
		}
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		{
			return "boolean"
		}
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		{
			return "string"
		}
	}
	return "object"
}

func methodParamsComment(desc *descriptor.MethodDescriptorProto, index int, types map[string]*descriptor.DescriptorProto, comments map[string]*descriptor.SourceCodeInfo_Location, indent string) h.HTML {
	name := strings.Split(desc.GetInputType(), ".")
	t := types[name[len(name)-1]]
	lines := make([]string, 0)
	lines = append(lines, "@static")
	lines = append(lines, "@typedef {Object} data")
	for i, field := range t.Field {
		p := getFieldPath(index, i)
		c := comments[p]
		comment := strings.TrimSpace(c.GetLeadingComments())
		if strings.HasPrefix(comment, field.GetName()) {
			lines = append(lines, "@property {"+getJSType(field)+"} "+comment)
		} else {
			lines = append(lines, "@property {"+getJSType(field)+"} "+field.GetName()+" "+comment)
		}
	}
	lines = append(lines, "@type {Object} headers request headers")
	lines = append(lines, "@returns {Promise} Promise object returns the data result")
	return h.HTML(strings.Join(lines, "\n"+indent+" "))
}

func buildArgsObject(desc *descriptor.MethodDescriptorProto, types map[string]*descriptor.DescriptorProto) string {
	name := strings.Split(desc.GetInputType(), ".")
	// log.Println(desc.GetInputType())
	t := types[name[len(name)-1]]
	a := "{"
	for _, field := range t.Field {
		s := getValueExtension(field, eproto.E_Default)
		if s != "" {
			a += fmt.Sprintf("%s = %s, ", field.GetName(), s)
			continue
		}
	}
	if len(a) == 1 {
		return "___unused" // create a placeholder value so eslint doesn't complain and any value is ignored
	}
	a = strings.TrimRight(a, ", ")
	a += "}"
	return a
}

func buildDataObject(desc *descriptor.MethodDescriptorProto, types map[string]*descriptor.DescriptorProto) string {
	name := strings.Split(desc.GetInputType(), ".")
	t := types[name[len(name)-1]]
	a := ""
	for _, field := range t.Field {
		if field.GetName() != "" {
			a += fmt.Sprintf("%s, ", field.GetName())
		}
	}
	a = strings.TrimRight(a, ", ")
	return a
}

func generate(in []byte) []byte {
	req := new(plugin.CodeGeneratorRequest)
	err := proto.Unmarshal(in, req)
	if err != nil {
		log.Fatalln(err)
	}
	resp := &plugin.CodeGeneratorResponse{
		File: make([]*plugin.CodeGeneratorResponse_File, 0),
	}
	tmpl := template.New("es6")
	tmpl = tmpl.Funcs(map[string]interface{}{
		"messageComment":      messageComment,
		"methodComment":       methodComment,
		"methodParamsComment": methodParamsComment,
		"buildArgsObject":     buildArgsObject,
		"buildDataObject":     buildDataObject,
	})
	tmpl, err = tmpl.Parse(es6template)
	if err != nil {
		log.Fatalln(err)
	}
	htmpl, err := template.New("header").Parse(headerTemplate)
	if err != nil {
		log.Fatalln(err)
	}
	date := time.Now().Format(time.ANSIC)
	var buf bytes.Buffer
	filesToGenerate := make(map[string]bool)
	for _, fn := range req.GetFileToGenerate() {
		filesToGenerate[fn] = true
	}
	for _, protofile := range req.ProtoFile {
		if filesToGenerate[protofile.GetName()] {
			filename := protofile.GetName()
			// log.Printf("incoming file %s\n", filename)
			buf.Reset()
			htmpl.Execute(&buf, map[string]string{
				"filename": filename,
				"date":     date,
			})
			pkg := protofile.GetPackage()
			if strings.Contains(pkg, ".") {
				tok := strings.Split(pkg, ".")
				pkg = tok[len(tok)-1]
			}
			header := buf.String()
			for sindex, service := range protofile.Service {
				buf.Reset()
				// ex: /api/v1.UserManager/Create
				apiprefix := "/api/" + pkg + "." + service.GetName()
				comments := extractComments(protofile)
				err := tmpl.Execute(&buf, map[string]interface{}{
					"header":       header,
					"apiprefix":    apiprefix,
					"filename":     filename,
					"service":      service,
					"serviceindex": sindex,
					"package":      pkg,
					"types":        extractTypes(protofile.MessageType),
					"comment":      strings.Replace(strings.TrimSpace(comments[getServicePath(sindex)].GetLeadingComments()), "\n", "\n//", -1),
					"comments":     comments,
				})
				if err != nil {
					log.Fatalln(err)
				}
				resp.File = append(resp.File, &plugin.CodeGeneratorResponse_File{
					Name:    strp(service.GetName() + ".js"),
					Content: strp(buf.String()),
				})
				// log.Println(buf.String())
			}
		}
	}
	// fmt.Fprintln(os.Stderr, util.Stringify(resp, true))
	out, err := proto.Marshal(resp)
	if err != nil {
		log.Fatalln(err)
	}
	return out
}

func main() {
	log.SetFlags(255)
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln(err)
	}
	ob := generate(b)
	os.Stdout.Write(ob)
}

const headerTemplate = `// Code generated by protoc-gen-es6rpc
// WARNING: DO NOT HAND EDIT THIS OR YOU WILL LOSE CHANGES
// Generated from {{ .filename }}
`

const es6template = `{{ .header }}

{{ $comments := .comments -}}
{{ $types := .types -}}
{{ $apiprefix := .apiprefix -}}
{{ $serviceindex := .serviceindex -}}

// {{ .comment }}
// @class
export class {{ .service.Name }} {
{{ range $i, $method := .service.Method }}
	{{ with $method -}}
	/**
	 * {{ methodComment $serviceindex $i $comments "\t*" }}
	 * {{ methodParamsComment $method $i $types $comments "\t *" }}
	 */
	static {{.Name}}({{ buildArgsObject $method $types }}, headers = {}) {
		return new Promise((resolve, reject) => {
			const data = { {{ buildDataObject $method $types }} };
			const xhr = new XMLHttpRequest();
			xhr.open('POST', '{{$apiprefix}}/{{.Name}}', true);
			xhr.setRequestHeader('Content-Type', 'application/json');
			if (headers) {
				Object.keys(headers).forEach(key => {
					xhr.setRequestHeader(key, headers[key]);
				});
			}
			xhr.onerror = () => reject(new Error(xhr.statusText));
			xhr.onload = () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					try {
						resolve(JSON.parse(xhr.response));
					} catch (ex) {
						reject(ex);
					}
				} else {
					reject(new Error(xhr.statusText));
				}
			};
			xhr.send(JSON.stringify(data));
		});
	}
	{{- end }}
{{- end }}
}
`
