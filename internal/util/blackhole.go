package util

type BlackHoleWriter struct{}

func (w *BlackHoleWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
