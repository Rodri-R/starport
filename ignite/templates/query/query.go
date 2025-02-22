package query

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/genny"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/plush/v4"

	"github.com/ignite/cli/ignite/pkg/gomodulepath"
	"github.com/ignite/cli/ignite/pkg/placeholder"
	"github.com/ignite/cli/ignite/pkg/xgenny"
	"github.com/ignite/cli/ignite/templates/field/plushhelpers"
)

//go:embed files/* files/**/*
var fs embed.FS

func Box(box packd.Walker, opts *Options, g *genny.Generator) error {
	if err := g.Box(box); err != nil {
		return err
	}
	ctx := plush.NewContext()
	ctx.Set("ModuleName", opts.ModuleName)
	ctx.Set("AppName", opts.AppName)
	ctx.Set("QueryName", opts.QueryName)
	ctx.Set("Description", opts.Description)
	ctx.Set("ModulePath", opts.ModulePath)
	ctx.Set("ReqFields", opts.ReqFields)
	ctx.Set("ResFields", opts.ResFields)
	ctx.Set("Paginated", opts.Paginated)

	plushhelpers.ExtendPlushContext(ctx)
	g.Transformer(xgenny.Transformer(ctx))
	g.Transformer(genny.Replace("{{appName}}", opts.AppName))
	g.Transformer(genny.Replace("{{moduleName}}", opts.ModuleName))
	g.Transformer(genny.Replace("{{queryName}}", opts.QueryName.Snake))
	return nil
}

// NewGenerator returns the generator to scaffold a empty query in a module
func NewGenerator(replacer placeholder.Replacer, opts *Options) (*genny.Generator, error) {
	var (
		g        = genny.New()
		template = xgenny.NewEmbedWalker(
			fs,
			"files/",
			opts.AppPath,
		)
	)

	g.RunFn(protoQueryModify(replacer, opts))
	g.RunFn(cliQueryModify(replacer, opts))

	return g, Box(template, opts, g)
}

func protoQueryModify(replacer placeholder.Replacer, opts *Options) genny.RunFn {
	return func(r *genny.Runner) error {
		path := filepath.Join(opts.AppPath, "proto", opts.AppName, opts.ModuleName, "query.proto")
		f, err := r.Disk.Find(path)
		if err != nil {
			return err
		}

		// if the query has request fields, they are appended to the rpc query
		var reqPath string
		for _, field := range opts.ReqFields {
			reqPath += "/"
			reqPath = filepath.Join(reqPath, fmt.Sprintf("{%s}", field.ProtoFieldName()))
		}

		// RPC service
		templateRPC := `// Queries a list of %[2]v items.
	rpc %[2]v(Query%[2]vRequest) returns (Query%[2]vResponse) {
		option (google.api.http).get = "/%[3]v/%[4]v/%[5]v%[6]v";
	}

%[1]v`
		appModulePath := gomodulepath.ExtractAppPath(opts.ModulePath)
		replacementRPC := fmt.Sprintf(
			templateRPC,
			Placeholder2,
			opts.QueryName.UpperCamel,
			appModulePath,
			opts.ModuleName,
			opts.QueryName.Snake,
			reqPath,
		)
		content := replacer.Replace(f.String(), Placeholder2, replacementRPC)

		// Fields for request
		var reqFields string
		for i, field := range opts.ReqFields {
			reqFields += fmt.Sprintf("  %s;\n", field.ProtoType(i+1))
		}
		if opts.Paginated {
			reqFields += fmt.Sprintf("cosmos.base.query.v1beta1.PageRequest pagination = %d;\n", len(opts.ReqFields)+1)
		}

		// Fields for response
		var resFields string
		for i, field := range opts.ResFields {
			resFields += fmt.Sprintf("  %s;\n", field.ProtoType(i+1))
		}
		if opts.Paginated {
			resFields += fmt.Sprintf("cosmos.base.query.v1beta1.PageResponse pagination = %d;\n", len(opts.ResFields)+1)
		}

		// Ensure custom types are imported
		protoImports := append(opts.ResFields.ProtoImports(), opts.ReqFields.ProtoImports()...)
		customFields := append(opts.ResFields.Custom(), opts.ReqFields.Custom()...)
		for _, f := range customFields {
			protoImports = append(protoImports,
				fmt.Sprintf("%[1]v/%[2]v/%[3]v.proto", opts.AppName, opts.ModuleName, f),
			)
		}
		for _, f := range protoImports {
			importModule := fmt.Sprintf(`
import "%[1]v";`, f)
			content = strings.ReplaceAll(content, importModule, "")

			replacementImport := fmt.Sprintf("%[1]v%[2]v", Placeholder, importModule)
			content = replacer.Replace(content, Placeholder, replacementImport)
		}

		// Messages
		templateMessages := `message Query%[2]vRequest {
%[3]v}

message Query%[2]vResponse {
%[4]v}

%[1]v`
		replacementMessages := fmt.Sprintf(
			templateMessages,
			Placeholder3,
			opts.QueryName.UpperCamel,
			reqFields,
			resFields,
		)
		content = replacer.Replace(content, Placeholder3, replacementMessages)

		newFile := genny.NewFileS(path, content)
		return r.File(newFile)
	}
}

func cliQueryModify(replacer placeholder.Replacer, opts *Options) genny.RunFn {
	return func(r *genny.Runner) error {
		path := filepath.Join(opts.AppPath, "x", opts.ModuleName, "client/cli/query.go")
		f, err := r.Disk.Find(path)
		if err != nil {
			return err
		}

		template := `cmd.AddCommand(Cmd%[2]v())

%[1]v`
		replacement := fmt.Sprintf(
			template,
			Placeholder,
			opts.QueryName.UpperCamel,
		)
		content := replacer.Replace(f.String(), Placeholder, replacement)

		newFile := genny.NewFileS(path, content)
		return r.File(newFile)
	}
}
