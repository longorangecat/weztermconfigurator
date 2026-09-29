package catalog

// DocURL returns the WezTerm documentation page for a top-level config option.
func DocURL(option string) string {
	return "https://wezterm.org/config/lua/config/" + option + ".html"
}
