package functions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = EchoFunction{}

func NewEchoFunction() function.Function {
	return &EchoFunction{}
}

type EchoFunction struct{}

func (f EchoFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "echo"
}

func (f EchoFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Parameters: []function.Parameter{
			function.StringParameter{},
		},
		Return: function.StringReturn{},
	}
}

func (f EchoFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var arg string

	resp.Diagnostics.Append(req.Arguments.Get(ctx, &arg)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, arg)...)
}
