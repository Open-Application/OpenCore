package liboc

var (
	localeRegistry = make(map[string]*Locale)
	current        = defaultLocal
)

type Locale struct {
	Locale                  string
	DeprecatedMessage       string
	DeprecatedMessageNoLink string
}

var defaultLocal = &Locale{
	Locale:                  "en_US",
	DeprecatedMessage:       "%s is deprecated in liboc %s and will be removed in liboc %s please checkout documentation for migration.",
	DeprecatedMessageNoLink: "%s is deprecated in liboc %s and will be removed in liboc %s.",
}

func Current() *Locale {
	return current
}

func Set(localeId string) bool {
	locale, loaded := localeRegistry[localeId]
	if !loaded {
		return false
	}
	current = locale
	return true
}