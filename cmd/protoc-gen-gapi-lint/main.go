package main

import (
	"errors"

	"github.com/jhump/protoreflect/desc"
	"github.com/moward/protoc-gen-gapi-lint/internal/lint"
	"github.com/moward/protoc-gen-gapi-lint/internal/lint/format"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

func NewFlagSet(config *lint.Config) *pflag.FlagSet {
	args := pflag.NewFlagSet("protoc-gen-gapi-lint", pflag.ExitOnError)
	args.StringVar(&config.Path, "config", "", "The linter config file.")
	args.StringVar(&config.OutputFormat, "output-format", "", "The format of the linting results.\nSupported formats include \"yaml\", \"json\",\"github\",\"summary\" table, and \"pretty\".\nYAML is the default.")
	args.StringVarP(&config.OutputPath, "output-path", "o", "", "The output file path.\nIf not given, the linting results will be printed out to STDOUT.")
	args.StringArrayVar(&config.EnabledRules, "enable-rule", nil, "Enable a rule with the given name.\nMay be specified multiple times.")
	args.StringArrayVar(&config.DisabledRules, "disable-rule", nil, "Disable a rule with the given name.\nMay be specified multiple times.")
	args.BoolVar(&config.IgnoreCommentDisables, "ignore-comment-disables", false, "If set to true, disable comments will be ignored.\nThis is helpful when strict enforcement of AIPs are necessary and\nproto definitions should not be able to disable checks.")
	args.BoolVar(&config.SetExitStatus, "set-exit-status", false, "If set to true, the exit status will be set to 1 if any linting errors are found.")
	args.StringArrayVar(&config.AllowedFiles, "allowed-files", nil, "A list of files to lint.\nIf not given, all files will be linted.")
	return args
}

func main() {
	config := &lint.Config{}

	// create the arguments
	args := NewFlagSet(config)
	// create the handler
	handler := protogen.Options{
		ParamFunc: args.Set,
	}

	handler.Run(func(plugin *protogen.Plugin) error {
		// Signals support for proto3 optional
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

		var collection []lint.Response

		linter, err := lint.New(config)
		if err != nil {
			return err
		}

		for _, file := range plugin.Files {
			if !file.Generate {
				continue
			}

			if len(config.AllowedFiles) > 0 {
				found := false
				for _, f := range config.AllowedFiles {
					if f == file.Desc.Path() {
						found = true
						break
					}
				}
				if !found {
					// skip the file
					continue
				}
			}

			fdesc, err := desc.WrapFile(file.Desc)
			if err != nil {
				return err
			}

			batch, err := linter.LintProtos(fdesc)
			if err != nil {
				return err
			}

			for _, item := range batch {
				if count := len(item.Problems); count == 0 {
					continue
				}

				collection = append(collection, item)
			}
		}

		if count := len(collection); count == 0 {
			return nil
		}

		writer, err := format.NewWriter(config.OutputPath)
		if err != nil {
			return err
		}
		// close the writer
		defer writer.Close()

		encoder := format.NewEncoder(writer, config.OutputFormat)
		// encode the collection
		if err := encoder.Encode(collection); err != nil {
			return err
		}

		if config.SetExitStatus && len(collection) > 0 {
			return errors.New("linting errors found")
		}

		return nil
	})
}
