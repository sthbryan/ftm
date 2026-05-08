package i18n

var Tr = T

func StatusText(state string) string {
	switch state {
	case "online":
		return T("online")
	case "offline":
		return T("offline")
	case "connecting":
		return T("connecting")
	case "error":
		return T("error")
	case "timeout":
		return T("timeout")
	case "starting":
		return T("starting")
	case "stopping":
		return T("stopping")
	default:
		return state
	}
}

func ProviderText(provider string) string {
	switch provider {
	case "pinggy":
		return T("provider_pinggy")
	case "serveo":
		return T("provider_serveo")
	case "cloudflared":
		return T("provider_cloudflared")
	case "tunnelmole":
		return T("provider_tunnelmole")
	case "localhostrun":
		return T("provider_localhostrun")
	default:
		return provider
	}
}

func NotificationText(key string, args ...string) string {
	msg := T(key)

	for i, arg := range args {
		placeholder := "{" + string(rune('0'+i)) + "}"
		msg = replaceAll(msg, placeholder, arg)
	}
	return msg
}

func replaceAll(s, old, new string) string {
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
