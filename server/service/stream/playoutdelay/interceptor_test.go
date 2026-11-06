package playoutdelay

import (
	"errors"
	"strconv"
	"testing"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

func TestNewFactoryValidatesDelay(t *testing.T) {
	t.Parallel()

	if _, err := NewFactory(0, 0); err != nil {
		t.Fatalf("NewFactory(0, 0) returned an error: %v", err)
	}

	if _, err := NewFactory(1<<12, 0); err == nil {
		t.Fatal("NewFactory accepted a delay larger than 12 bits")
	}

	if _, err := NewFactory(2, 1); err == nil {
		t.Fatal("NewFactory accepted a minimum delay larger than the maximum")
	}
}

func TestInterceptorUsesNegotiatedExtensionID(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}

	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	for _, id := range []int{5, 12} {
		id := id
		t.Run("id_"+strconv.Itoa(id), func(t *testing.T) {
			t.Parallel()

			called := false
			writer := inter.BindLocalStream(&interceptor.StreamInfo{
				RTPHeaderExtensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: id}},
			}, interceptor.RTPWriterFunc(func(
				_ *rtp.Header,
				payload []byte,
				_ interceptor.Attributes,
			) (int, error) {
				called = true
				return len(payload), nil
			}))

			header := &rtp.Header{}
			payload := []byte{1, 2, 3}
			n, writeErr := writer.Write(header, payload, nil)
			if writeErr != nil {
				t.Fatalf("write RTP packet: %v", writeErr)
			}
			if !called {
				t.Fatal("downstream writer was not called")
			}
			if n != len(payload) {
				t.Fatalf("write returned %d bytes, want %d", n, len(payload))
			}
			if header.ExtensionProfile != rtp.ExtensionProfileOneByte {
				t.Fatalf("profile = %#x, want one-byte %#x", header.ExtensionProfile, rtp.ExtensionProfileOneByte)
			}

			raw := header.GetExtension(uint8(id))
			if len(raw) != 3 {
				t.Fatalf("extension payload length = %d, want 3", len(raw))
			}

			var delay rtp.PlayoutDelayExtension
			if err := delay.Unmarshal(raw); err != nil {
				t.Fatalf("unmarshal extension: %v", err)
			}
			if delay.MinDelay != 0 || delay.MaxDelay != 0 {
				t.Fatalf("delay = {%d, %d}, want {0, 0}", delay.MinDelay, delay.MaxDelay)
			}
		})
	}
}

func TestInterceptorBypassesUnnegotiatedStreams(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}
	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	tests := []struct {
		name       string
		extensions []interceptor.RTPHeaderExtension
	}{
		{name: "none"},
		{name: "other URI", extensions: []interceptor.RTPHeaderExtension{{URI: "urn:example:other", ID: 5}}},
		{name: "zero ID", extensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 0}}},
		{name: "ID above two-byte range", extensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 256}}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			called := false
			writer := inter.BindLocalStream(&interceptor.StreamInfo{
				RTPHeaderExtensions: test.extensions,
			}, interceptor.RTPWriterFunc(func(
				_ *rtp.Header,
				payload []byte,
				_ interceptor.Attributes,
			) (int, error) {
				called = true
				return len(payload), nil
			}))

			header := &rtp.Header{}
			if _, err := writer.Write(header, []byte{1}, nil); err != nil {
				t.Fatalf("write RTP packet: %v", err)
			}
			if !called {
				t.Fatal("downstream writer was not called")
			}
			if header.Extension {
				t.Fatal("unnegotiated stream received an RTP header extension")
			}
		})
	}
}

func TestInterceptorUsesTwoByteProfileForHighNegotiatedIDs(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}
	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	tests := []struct {
		name       string
		extensions []interceptor.RTPHeaderExtension
		playoutID  uint8
	}{
		{
			name:       "playout delay ID is high",
			extensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 16}},
			playoutID:  16,
		},
		{
			name: "another negotiated ID is high",
			extensions: []interceptor.RTPHeaderExtension{
				{URI: URI, ID: 5},
				{URI: "urn:example:other", ID: 16},
			},
			playoutID: 5,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			writer := inter.BindLocalStream(&interceptor.StreamInfo{
				RTPHeaderExtensions: test.extensions,
			}, interceptor.RTPWriterFunc(func(
				_ *rtp.Header,
				payload []byte,
				_ interceptor.Attributes,
			) (int, error) {
				return len(payload), nil
			}))

			header := &rtp.Header{}
			if _, err := writer.Write(header, []byte{1}, nil); err != nil {
				t.Fatalf("write RTP packet: %v", err)
			}
			if header.ExtensionProfile != rtp.ExtensionProfileTwoByte {
				t.Fatalf("profile = %#x, want two-byte %#x", header.ExtensionProfile, rtp.ExtensionProfileTwoByte)
			}
			if raw := header.GetExtension(test.playoutID); len(raw) != 3 {
				t.Fatalf("extension payload length = %d, want 3", len(raw))
			}
		})
	}
}

func TestInterceptorUpgradesExistingOneByteProfile(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}
	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	writer := inter.BindLocalStream(&interceptor.StreamInfo{
		RTPHeaderExtensions: []interceptor.RTPHeaderExtension{
			{URI: "urn:example:existing", ID: 3},
			{URI: URI, ID: 16},
		},
	}, interceptor.RTPWriterFunc(func(
		_ *rtp.Header,
		payload []byte,
		_ interceptor.Attributes,
	) (int, error) {
		return len(payload), nil
	}))

	header := &rtp.Header{}
	if err := header.SetExtension(3, []byte{0xAA}); err != nil {
		t.Fatalf("set existing extension: %v", err)
	}
	if _, err := writer.Write(header, []byte{1}, nil); err != nil {
		t.Fatalf("write RTP packet: %v", err)
	}
	if header.ExtensionProfile != rtp.ExtensionProfileTwoByte {
		t.Fatalf("profile = %#x, want two-byte %#x", header.ExtensionProfile, rtp.ExtensionProfileTwoByte)
	}
	if raw := header.GetExtension(3); len(raw) != 1 || raw[0] != 0xAA {
		t.Fatalf("existing extension was not preserved: %v", raw)
	}
	if raw := header.GetExtension(16); len(raw) != 3 {
		t.Fatalf("playout extension payload length = %d, want 3", len(raw))
	}
}

func TestInterceptorDowngradesCompatibleTwoByteProfile(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}
	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	writer := inter.BindLocalStream(&interceptor.StreamInfo{
		RTPHeaderExtensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 5}},
	}, interceptor.RTPWriterFunc(func(
		_ *rtp.Header,
		payload []byte,
		_ interceptor.Attributes,
	) (int, error) {
		return len(payload), nil
	}))

	header := &rtp.Header{
		Extension:        true,
		ExtensionProfile: rtp.ExtensionProfileTwoByte,
	}
	if err := header.SetExtension(3, []byte{0xBB}); err != nil {
		t.Fatalf("set existing extension: %v", err)
	}
	if _, err := writer.Write(header, []byte{1}, nil); err != nil {
		t.Fatalf("write RTP packet: %v", err)
	}
	if header.ExtensionProfile != rtp.ExtensionProfileOneByte {
		t.Fatalf("profile = %#x, want one-byte %#x", header.ExtensionProfile, rtp.ExtensionProfileOneByte)
	}
	if raw := header.GetExtension(3); len(raw) != 1 || raw[0] != 0xBB {
		t.Fatalf("existing extension was not preserved: %v", raw)
	}
	if raw := header.GetExtension(5); len(raw) != 3 {
		t.Fatalf("playout extension payload length = %d, want 3", len(raw))
	}
}

func TestInterceptorDoesNotMutateHeaderWhenProfileConversionFails(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}
	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	called := false
	writer := inter.BindLocalStream(&interceptor.StreamInfo{
		RTPHeaderExtensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 5}},
	}, interceptor.RTPWriterFunc(func(
		_ *rtp.Header,
		payload []byte,
		_ interceptor.Attributes,
	) (int, error) {
		called = true
		return len(payload), nil
	}))

	header := &rtp.Header{
		Extension:        true,
		ExtensionProfile: rtp.ExtensionProfileTwoByte,
	}
	if err := header.SetExtension(16, []byte{0xCC}); err != nil {
		t.Fatalf("set existing extension: %v", err)
	}
	if _, err := writer.Write(header, []byte{1}, nil); err == nil {
		t.Fatal("write succeeded despite an incompatible one-byte profile conversion")
	}
	if called {
		t.Fatal("downstream writer was called after profile conversion failed")
	}
	if header.ExtensionProfile != rtp.ExtensionProfileTwoByte {
		t.Fatalf("profile changed to %#x after failure", header.ExtensionProfile)
	}
	if raw := header.GetExtension(16); len(raw) != 1 || raw[0] != 0xCC {
		t.Fatalf("existing extension changed after failure: %v", raw)
	}
	if header.GetExtension(5) != nil {
		t.Fatal("playout extension was committed after failure")
	}
}

func TestInterceptorKeepsStreamIDsIndependent(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}
	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	baseWriter := interceptor.RTPWriterFunc(func(
		_ *rtp.Header,
		payload []byte,
		_ interceptor.Attributes,
	) (int, error) {
		return len(payload), nil
	})
	writer5 := inter.BindLocalStream(&interceptor.StreamInfo{
		RTPHeaderExtensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 5}},
	}, baseWriter)
	writer12 := inter.BindLocalStream(&interceptor.StreamInfo{
		RTPHeaderExtensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 12}},
	}, baseWriter)

	header5 := &rtp.Header{}
	header12 := &rtp.Header{}
	if _, err := writer5.Write(header5, []byte{1}, nil); err != nil {
		t.Fatalf("write ID 5 stream: %v", err)
	}
	if _, err := writer12.Write(header12, []byte{1}, nil); err != nil {
		t.Fatalf("write ID 12 stream: %v", err)
	}
	if len(header5.GetExtension(5)) != 3 || header5.GetExtension(12) != nil {
		t.Fatalf("ID 5 stream extensions are incorrect: IDs %v", header5.GetExtensionIDs())
	}
	if len(header12.GetExtension(12)) != 3 || header12.GetExtension(5) != nil {
		t.Fatalf("ID 12 stream extensions are incorrect: IDs %v", header12.GetExtensionIDs())
	}
}

func TestInterceptorPropagatesErrors(t *testing.T) {
	t.Parallel()

	factory, err := NewFactory(0, 0)
	if err != nil {
		t.Fatalf("create factory: %v", err)
	}
	inter, err := factory.NewInterceptor("test")
	if err != nil {
		t.Fatalf("create interceptor: %v", err)
	}

	downstreamErr := errors.New("downstream failure")
	writer := inter.BindLocalStream(&interceptor.StreamInfo{
		RTPHeaderExtensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 5}},
	}, interceptor.RTPWriterFunc(func(
		_ *rtp.Header,
		_ []byte,
		_ interceptor.Attributes,
	) (int, error) {
		return 7, downstreamErr
	}))

	n, err := writer.Write(&rtp.Header{}, []byte{1}, nil)
	if n != 7 || !errors.Is(err, downstreamErr) {
		t.Fatalf("downstream result = (%d, %v), want (7, %v)", n, err, downstreamErr)
	}

	called := false
	nilHeaderWriter := inter.BindLocalStream(&interceptor.StreamInfo{
		RTPHeaderExtensions: []interceptor.RTPHeaderExtension{{URI: URI, ID: 5}},
	}, interceptor.RTPWriterFunc(func(
		_ *rtp.Header,
		_ []byte,
		_ interceptor.Attributes,
	) (int, error) {
		called = true
		return 0, nil
	}))
	if _, err := nilHeaderWriter.Write(nil, []byte{1}, nil); !errors.Is(err, errNilRTPHeader) {
		t.Fatalf("nil header error = %v, want %v", err, errNilRTPHeader)
	}
	if called {
		t.Fatal("downstream writer was called for a nil RTP header")
	}
}
