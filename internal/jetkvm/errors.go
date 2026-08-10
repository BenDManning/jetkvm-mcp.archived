package jetkvm

import "errors"

var (
	ErrAuthentication              = errors.New("JetKVM authentication failed")
	ErrDeviceUnreachable           = errors.New("JetKVM device unreachable")
	ErrInvalidResponse             = errors.New("invalid JetKVM response")
	ErrProtocol                    = errors.New("JetKVM protocol error")
	ErrUnknownDevice               = errors.New("unknown JetKVM device")
	ErrUnknownWakeTarget           = errors.New("unknown Wake-on-LAN target")
	ErrRPCMethodUnavailable        = errors.New("JetKVM RPC method unavailable")
	ErrUnsolicitedRPC              = errors.New("unsolicited JetKVM RPC event")
	ErrSessionClosed               = errors.New("JetKVM session closed")
	ErrUnsupportedInput            = errors.New("unsupported JetKVM input")
	ErrMediaPath                   = errors.New("invalid media path")
	ErrMediaDirectoryNotConfigured = errors.New("media directory is not configured")
	ErrVideoBusy                   = errors.New("JetKVM video capture already in progress")
	ErrDecoderUnavailable          = errors.New("FFmpeg decoder is unavailable")
)
