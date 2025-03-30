// Package mjpeg implements a simple MJPEG streamer.
//
// Stream objects implement the http.Handler interface, allowing to use them with the net/http package like so:
//
//	stream = mjpeg.NewStream()
//	http.Handle("/camera", stream)
//
// Then push new JPEG frames to the connected clients using stream.UpdateJPEG().
package mjpeg

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Stream represents a single video feed.
type Stream struct {
	start         time.Time
	m             map[chan []byte]bool
	frame         []byte
	lock          sync.Mutex
	FrameInterval time.Duration
}

const boundaryWord = "MJPEGBOUNDARY"
const headerf = "\r\n" +
	"--" + boundaryWord + "\r\n" +
	"Content-Type: image/jpeg\r\n" +
	"Content-Length: %d\r\n" +
	"X-Timestamp: %d.%d\r\n" +
	"\r\n"

// ServeHTTP responds to HTTP requests with the MJPEG stream, implementing the http.Handler interface.
func (s *Stream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("Stream:", r.RemoteAddr, "connected")
	w.Header().Add("Content-Type", "multipart/x-mixed-replace;boundary="+boundaryWord)

	c := make(chan []byte)
	s.lock.Lock()
	s.m[c] = true
	s.lock.Unlock()
	s.start = time.Now()

	for {
		time.Sleep(s.FrameInterval)
		b := <-c
		_, err := w.Write(b)
		if err != nil {
			slog.Error("Stream:%s write error %s", r.RemoteAddr, err.Error())
			break
		}
	}

	s.lock.Lock()
	delete(s.m, c)
	s.lock.Unlock()
	slog.Info("Stream:", r.RemoteAddr, "disconnected")
}

// UpdateJPEG pushes a new JPEG frame onto the clients.
func (s *Stream) UpdateJPEG(jpeg []byte) {
	if len(jpeg) == 0 {
		return
	}
	elapsed := time.Since(s.start)
	s.updateFrame(jpeg, elapsed)

	s.lock.Lock()
	for c := range s.m {
		// Select to skip streams which are sleeping to drop frames.
		// This might need more thought.
		select {
		case c <- s.frame:
		default:
		}
	}
	s.lock.Unlock()
}

// NewStream initializes and returns a new Stream.
func NewStream() *Stream {
	return &Stream{
		m:             make(map[chan []byte]bool),
		frame:         make([]byte, len(headerf)),
		FrameInterval: 50 * time.Millisecond,
	}
}

func (s *Stream) updateFrame(jpeg []byte, elapsed time.Duration) {
	header := s.frameHeader(jpeg, elapsed)
	if len(s.frame) < len(jpeg)+len(header) {
		s.frame = make([]byte, (len(jpeg)+len(header))*2)
	}

	copy(s.frame, header)
	copy(s.frame[len(header):], jpeg)
}

func (s *Stream) frameHeader(jpeg []byte, elapsed time.Duration) string {
	sec := int64(elapsed.Seconds())
	usec := int64(elapsed.Microseconds() % 1e6)
	return fmt.Sprintf(headerf, len(jpeg), sec, usec)
}
