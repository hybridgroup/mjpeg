package mjpeg

import (
	"testing"
	"time"
)

func TestFrameHeader(t *testing.T) {
	// Create a new Stream instance
	stream := NewStream()

	// Simulate sending a JPEG frame to the stream
	jpegFrame := []byte("test_frame")
	elapsed := time.Duration(15535 * time.Millisecond)
	header := stream.frameHeader(jpegFrame, elapsed)

	// Check if the header is correctly formatted
	expected := "\r\n" +
		"--MJPEGBOUNDARY\r\n" +
		"Content-Type: image/jpeg\r\n" +
		"Content-Length: 10\r\n" +
		"X-Timestamp: 15.535000\r\n" +
		"\r\n"

	if header != expected {
		t.Errorf("Expected header %s, got %s", expected, header)
	}
}
