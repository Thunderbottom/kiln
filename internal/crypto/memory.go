package crypto

import (
	"crypto/rand"
	"errors"
	"runtime"

	"golang.org/x/sys/unix"
)

type SecureBuffer struct {
	data []byte
	size int
}

func NewSecureBuffer(size int) (*SecureBuffer, error) {
	data := make([]byte, size)

	if err := mlock(data); err != nil {
		return nil, err
	}

	sb := &SecureBuffer{
		data: data,
		size: size,
	}

	runtime.SetFinalizer(sb, (*SecureBuffer).Close)
	return sb, nil
}

func (sb *SecureBuffer) Write(data []byte) error {
	if len(data) > sb.size {
		return ErrBufferTooSmall
	}

	copy(sb.data, data)
	return nil
}

func (sb *SecureBuffer) Read() []byte {
	result := make([]byte, sb.size)
	copy(result, sb.data)
	return result
}

func (sb *SecureBuffer) Clear() {
	if sb.data != nil {
		for i := range sb.data {
			sb.data[i] = 0
		}

		rand.Read(sb.data)

		for i := range sb.data {
			sb.data[i] = 0
		}
	}
}

func (sb *SecureBuffer) Close() error {
	if sb.data != nil {
		sb.Clear()
		if err := munlock(sb.data); err != nil {
			return err
		}
		sb.data = nil
	}
	runtime.SetFinalizer(sb, nil)
	return nil
}

func mlock(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return unix.Mlock(data)
}

func munlock(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return unix.Munlock(data)
}

var ErrBufferTooSmall = errors.New("buffer too small")
