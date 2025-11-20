package arguments

import "github.com/capitalrx/grafana/pkg/build/daggerbuild/pipeline"

func Join(f ...[]pipeline.Argument) []pipeline.Argument {
	r := []pipeline.Argument{}
	for _, v := range f {
		r = append(r, v...)
	}

	return r
}
