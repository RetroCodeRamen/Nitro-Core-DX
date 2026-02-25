package corelx

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CompileFile(sourcePath string, opts *CompileOptions) (*CompileResult, error) {
	return CompileProject(sourcePath, opts)
}

func (s *Service) CompileSource(source, sourcePath string, opts *CompileOptions) (*CompileResult, error) {
	return CompileSource(source, sourcePath, opts)
}

func (s *Service) CompileBundleFile(sourcePath string, opts *CompileOptions) (CompileBundle, *CompileResult, error) {
	res, err := CompileProject(sourcePath, opts)
	return BuildCompileBundle(res), res, err
}

func (s *Service) CompileBundleSource(source, sourcePath string, opts *CompileOptions) (CompileBundle, *CompileResult, error) {
	res, err := CompileSource(source, sourcePath, opts)
	return BuildCompileBundle(res), res, err
}
