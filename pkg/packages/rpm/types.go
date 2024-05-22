package rpm

const (
	compressionXZ   = "xz"
	compressionGzip = "gzip"
)

var supportedRPMCompressionTypes = []string{
	compressionXZ,
	compressionGzip,
}
