package playoutdelay

import (
	"errors"
	"fmt"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

const (
	// URI identifies the WebRTC playout-delay RTP header extension.
	URI = "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay"

	maxOneByteExtensionID = 14
	maxTwoByteExtensionID = 255
)

var (
	errNilRTPHeader             = errors.New("playout-delay: RTP header is nil")
	errUnsupportedHeaderProfile = errors.New("playout-delay: unsupported RTP header extension profile")
)

// Factory creates interceptors that add a fixed playout-delay value to every
// packet of an outgoing RTP stream that negotiated the extension.
type Factory struct {
	payload []byte
}

// NewFactory validates and marshals the delay bounds once for reuse by all
// packets. Delay values are expressed in 10 ms units by the extension format.
func NewFactory(minDelay, maxDelay uint16) (*Factory, error) {
	if minDelay > maxDelay {
		return nil, fmt.Errorf("invalid playout-delay range: minimum %d exceeds maximum %d", minDelay, maxDelay)
	}

	payload, err := (rtp.PlayoutDelayExtension{
		MinDelay: minDelay,
		MaxDelay: maxDelay,
	}).Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal playout-delay extension: %w", err)
	}

	return &Factory{payload: payload}, nil
}

// NewInterceptor implements interceptor.Factory.
func (f *Factory) NewInterceptor(_ string) (interceptor.Interceptor, error) {
	return &outboundInterceptor{
		payload: append([]byte(nil), f.payload...),
	}, nil
}

type outboundInterceptor struct {
	interceptor.NoOp
	payload []byte
}

// BindLocalStream uses the negotiated URI/ID mapping for this specific RTP
// stream. If playout-delay was not negotiated, the original writer is left
// untouched.
func (i *outboundInterceptor) BindLocalStream(
	info *interceptor.StreamInfo,
	writer interceptor.RTPWriter,
) interceptor.RTPWriter {
	extensionID, profile, ok := negotiatedExtension(info)
	if !ok {
		return writer
	}

	return interceptor.RTPWriterFunc(func(
		header *rtp.Header,
		payload []byte,
		attributes interceptor.Attributes,
	) (int, error) {
		if header == nil {
			return 0, errNilRTPHeader
		}

		if err := setExtension(header, extensionID, profile, i.payload); err != nil {
			return 0, fmt.Errorf("set playout-delay RTP extension: %w", err)
		}

		return writer.Write(header, payload, attributes)
	})
}

func negotiatedExtension(info *interceptor.StreamInfo) (uint8, uint16, bool) {
	if info == nil {
		return 0, 0, false
	}

	profile := uint16(rtp.ExtensionProfileOneByte)
	extensionID := 0

	for _, extension := range info.RTPHeaderExtensions {
		if extension.ID < 1 || extension.ID > maxTwoByteExtensionID {
			return 0, 0, false
		}

		if extension.ID > maxOneByteExtensionID {
			profile = rtp.ExtensionProfileTwoByte
		}

		if extension.URI == URI {
			extensionID = extension.ID
		}
	}

	if extensionID == 0 {
		return 0, 0, false
	}

	return uint8(extensionID), profile, true
}

func setExtension(header *rtp.Header, id uint8, profile uint16, payload []byte) error {
	updated := header.Clone()

	if !updated.Extension {
		updated.Extension = true
		updated.ExtensionProfile = profile
	} else {
		switch updated.ExtensionProfile {
		case rtp.ExtensionProfileOneByte:
			if profile == rtp.ExtensionProfileTwoByte {
				if err := updated.SetExtensionWithProfile(id, payload, rtp.ExtensionProfileTwoByte); err != nil {
					return err
				}
				*header = updated

				return nil
			}
		case rtp.ExtensionProfileTwoByte:
			if profile == rtp.ExtensionProfileOneByte {
				if err := updated.SetExtensionWithProfile(id, payload, rtp.ExtensionProfileOneByte); err != nil {
					return err
				}
				*header = updated

				return nil
			}
		default:
			return errUnsupportedHeaderProfile
		}
	}

	if err := updated.SetExtension(id, payload); err != nil {
		return err
	}

	*header = updated

	return nil
}
