package rpm

const (
	compressionXZ   = "xz"
	compressionGzip = "gzip"
	compressionZstd = "zstd"
)

var supportedRPMCompressionTypes = []string{
	compressionXZ,
	compressionGzip,
	compressionZstd,
}
