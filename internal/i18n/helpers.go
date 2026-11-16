package i18n

var Tr = T

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
	case "bore":
		return T("provider_bore")
	default:
		return provider
	}
}
