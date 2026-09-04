package catalog

// ActionDef describes one KeyAssignment constructor (Appendix C).
type ActionDef struct {
	Name, Doc string
	Arg       *Field // Arg nil = unit variant; BareOK = `act.Name` valid with all-default args
	BareOK    bool
}

// act builds a unit action; actArg a parameterized one.
func act(name, doc string) ActionDef { return ActionDef{Name: name, Doc: doc} }
func actArg(name, doc string, f Field) ActionDef {
	return ActionDef{Name: name, Doc: doc, Arg: &f}
}

var actionDefs = []ActionDef{
	actArg("SpawnTab", "Create a new tab in the given domain", Field{Name: "SpawnTab", Kind: Union, Enum: SpawnTabDomainNames, Fields: spawnTabDomainFields, Default: "CurrentPaneDomain"}),
	act("SpawnWindow", "Create a new window with the default tab"),
	act("ToggleFullScreen", "Toggle full screen"),
	act("ToggleAlwaysOnTop", "Toggle stay-on-top (macOS)"),
	act("ToggleAlwaysOnBottom", "Toggle stay-behind (macOS)"),
	actArg("SetWindowLevel", "Set window level (macOS)", Field{Name: "SetWindowLevel", Kind: Enum, Enum: []string{"AlwaysOnBottom", "Normal", "AlwaysOnTop"}}),
	actArg("CopyTo", "Copy the selection to a buffer", Field{Name: "CopyTo", Kind: Enum, Enum: []string{"Clipboard", "PrimarySelection", "ClipboardAndPrimarySelection"}}),
	actArg("CopyTextTo", "Copy literal text to a buffer", Field{Name: "CopyTextTo", Kind: Struct, Fields: []Field{
		{Name: "text", Kind: String, Required: true},
		{Name: "destination", Kind: Enum, Enum: []string{"Clipboard", "PrimarySelection", "ClipboardAndPrimarySelection"}, Required: true},
	}}),
	actArg("PasteFrom", "Paste from a buffer", Field{Name: "PasteFrom", Kind: Enum, Enum: []string{"Clipboard", "PrimarySelection"}}),
	actArg("ActivateTabRelative", "Activate tab by offset, wrapping", Field{Name: "ActivateTabRelative", Kind: Int}),
	actArg("ActivateTabRelativeNoWrap", "Activate tab by offset, no wrap", Field{Name: "ActivateTabRelativeNoWrap", Kind: Int}),
	act("IncreaseFontSize", "Increase font size 10%"),
	act("DecreaseFontSize", "Decrease font size 10%"),
	act("ResetFontSize", "Reset font size"),
	act("ResetFontAndWindowSize", "Reset font and window size"),
	actArg("ActivateTab", "Activate tab by zero-based index (negative counts from the end)", Field{Name: "ActivateTab", Kind: Int}),
	act("ActivateLastTab", "Activate the previously active tab"),
	actArg("SendString", "Send text/escape sequences to the pane", Field{Name: "SendString", Kind: String}),
	actArg("SendKey", "Send a synthetic key press", Field{Name: "SendKey", Kind: Struct, Fields: []Field{
		{Name: "key", Kind: String, Required: true},
		{Name: "mods", Kind: Flags, Enum: modFlags, EmptyToken: "NONE"},
	}}),
	act("Nop", "Swallow the key press"),
	act("DisableDefaultAssignment", "Remove the matching default assignment"),
	act("Hide", "Hide/minimize the window"),
	act("Show", "Show the window"),
	actArg("CloseCurrentTab", "Close the tab, optionally confirming", Field{Name: "CloseCurrentTab", Kind: Struct, Fields: []Field{
		{Name: "confirm", Kind: Bool, Required: true},
	}}),
	act("ReloadConfiguration", "Reload the configuration"),
	actArg("MoveTabRelative", "Move tab by offset", Field{Name: "MoveTabRelative", Kind: Int}),
	actArg("MoveTab", "Move tab to index", Field{Name: "MoveTab", Kind: Int, Min: ptr(0)}),
	actArg("ScrollByPage", "Scroll N pages (negative = up)", Field{Name: "ScrollByPage", Kind: Float}),
	actArg("ScrollByLine", "Scroll N lines", Field{Name: "ScrollByLine", Kind: Int}),
	act("ScrollByCurrentEventWheelDelta", "Scroll by the wheel delta (mouse)"),
	actArg("ScrollToPrompt", "Scroll to previous/next prompt zone", Field{Name: "ScrollToPrompt", Kind: Int}),
	act("ScrollToTop", "Scroll to top"),
	act("ScrollToBottom", "Scroll to bottom"),
	act("ShowTabNavigator", "Show the tab navigator"),
	act("ShowDebugOverlay", "Show the debug overlay"),
	act("HideApplication", "Hide the application (macOS)"),
	act("QuitApplication", "Quit WezTerm"),
	actArg("SpawnCommandInNewTab", "Spawn a command in a new tab", Field{Name: "SpawnCommandInNewTab", Kind: Struct, Fields: SpawnCommandFields}),
	actArg("SpawnCommandInNewWindow", "Spawn a command in a new window", Field{Name: "SpawnCommandInNewWindow", Kind: Struct, Fields: SpawnCommandFields}),
	actArg("SplitHorizontal", "Split; new pane on the right", Field{Name: "SplitHorizontal", Kind: Struct, Fields: SpawnCommandFields}),
	actArg("SplitVertical", "Split; new pane below", Field{Name: "SplitVertical", Kind: Struct, Fields: SpawnCommandFields}),
	act("ShowLauncher", "Show the launcher menu"),
	actArg("ShowLauncherArgs", "Launcher scoped by flags", Field{Name: "ShowLauncherArgs", Kind: Struct, Fields: []Field{
		{Name: "flags", Kind: Flags, Enum: []string{"FUZZY", "TABS", "LAUNCH_MENU_ITEMS", "DOMAINS", "KEY_ASSIGNMENTS", "WORKSPACES", "COMMANDS"}, EmptyToken: "", Required: true},
		{Name: "title", Kind: String},
		{Name: "help_text", Kind: String},
		{Name: "fuzzy_help_text", Kind: String},
		{Name: "alphabet", Kind: String},
	}}),
	actArg("ClearScrollback", "Clear scrollback (and optionally viewport)", Field{Name: "ClearScrollback", Kind: Enum, Enum: []string{"ScrollbackOnly", "ScrollbackAndViewport"}, Default: "ScrollbackOnly"}),
	actArg("Search", "Open the search overlay with a pattern", Field{Name: "Search", Kind: Union, Enum: []string{"CurrentSelectionOrEmptyString", "CaseSensitiveString", "CaseInSensitiveString", "CaseSmartString", "Regex"}, Fields: []Field{
		{Name: "CaseSensitiveString", Kind: Union, Scalar: true, Fields: []Field{{Name: "CaseSensitiveString", Kind: String}}},
		{Name: "CaseInSensitiveString", Kind: Union, Scalar: true, Fields: []Field{{Name: "CaseInSensitiveString", Kind: String}}},
		{Name: "CaseSmartString", Kind: Union, Scalar: true, Fields: []Field{{Name: "CaseSmartString", Kind: String}}},
		{Name: "Regex", Kind: Union, Scalar: true, Fields: []Field{{Name: "Regex", Kind: String}}},
	}}),
	act("ActivateCopyMode", "Enter copy mode"),
	actArg("SelectTextAtMouseCursor", "Start a selection at the mouse cursor", Field{Name: "SelectTextAtMouseCursor", Kind: Enum, Enum: []string{"Cell", "Word", "Line", "SemanticZone", "Block"}}),
	actArg("ExtendSelectionToMouseCursor", "Extend the selection to the mouse cursor", Field{Name: "ExtendSelectionToMouseCursor", Kind: Enum, Enum: []string{"Cell", "Word", "Line", "SemanticZone", "Block"}}),
	act("OpenLinkAtMouseCursor", "Open the link under the mouse"),
	act("ClearSelection", "Clear the selection"),
	actArg("CompleteSelection", "Finish the selection and copy", Field{Name: "CompleteSelection", Kind: Enum, Enum: []string{"Clipboard", "PrimarySelection", "ClipboardAndPrimarySelection"}}),
	actArg("CompleteSelectionOrOpenLinkAtMouseCursor", "Finish selection, else open link", Field{Name: "CompleteSelectionOrOpenLinkAtMouseCursor", Kind: Enum, Enum: []string{"Clipboard", "PrimarySelection", "ClipboardAndPrimarySelection"}}),
	act("StartWindowDrag", "Drag the window (mouse)"),
	actArg("AdjustPaneSize", "Resize the pane in a direction", Field{Name: "AdjustPaneSize", Kind: Struct, Tuple: true, Fields: []Field{
		{Name: "direction", Kind: Enum, Enum: []string{"Left", "Right", "Up", "Down"}, Required: true},
		{Name: "amount", Kind: Int, Min: ptr(1), Required: true},
	}}),
	actArg("ActivatePaneDirection", "Focus the adjacent pane", Field{Name: "ActivatePaneDirection", Kind: Enum, Enum: []string{"Up", "Down", "Left", "Right", "Next", "Prev"}}),
	actArg("ActivatePaneByIndex", "Activate pane by index", Field{Name: "ActivatePaneByIndex", Kind: Int, Min: ptr(0)}),
	act("TogglePaneZoomState", "Toggle pane zoom"),
	actArg("SetPaneZoomState", "Set pane zoom on/off", Field{Name: "SetPaneZoomState", Kind: Bool}),
	actArg("CloseCurrentPane", "Close the pane, optionally confirming", Field{Name: "CloseCurrentPane", Kind: Struct, Fields: []Field{
		{Name: "confirm", Kind: Bool, Required: true},
	}}),
	actArg("EmitEvent", "Emit a custom event (wezterm.on)", Field{Name: "EmitEvent", Kind: String}),
	act("QuickSelect", "Quick select mode"),
	actArg("QuickSelectArgs", "Quick select with overrides", Field{Name: "QuickSelectArgs", Kind: Struct, Fields: []Field{
		{Name: "alphabet", Kind: String},
		{Name: "patterns", Kind: StringList},
		{Name: "action", Kind: Lua},
		{Name: "skip_action_on_paste", Kind: Bool},
		{Name: "label", Kind: String},
		{Name: "scope_lines", Kind: Int, Min: ptr(0)},
	}}),
	actArg("Multiple", "Run several actions in sequence", Field{Name: "Multiple", Kind: List, ItemKind: Action}),
	actArg("SwitchToWorkspace", "Switch to (or create) a workspace", Field{Name: "SwitchToWorkspace", Kind: Struct, Fields: []Field{
		{Name: "name", Kind: String},
		{Name: "spawn", Kind: Struct, Fields: SpawnCommandFields},
	}}),
	actArg("SwitchWorkspaceRelative", "Switch workspace by offset", Field{Name: "SwitchWorkspaceRelative", Kind: Int}),
	actArg("ActivateKeyTable", "Activate a named key table", Field{Name: "ActivateKeyTable", Kind: Struct, Fields: []Field{
		{Name: "name", Kind: String, Required: true},
		{Name: "timeout_milliseconds", Kind: Int, Min: ptr(0)},
		{Name: "one_shot", Kind: Bool, Default: true},
		{Name: "replace_current", Kind: Bool},
		{Name: "until_unknown", Kind: Bool},
		{Name: "prevent_fallback", Kind: Bool},
	}}),
	act("PopKeyTable", "Pop the current key table"),
	act("ClearKeyTableStack", "Clear the key table stack"),
	actArg("DetachDomain", "Detach a multiplexing domain", Field{Name: "DetachDomain", Kind: Union, Enum: SpawnTabDomainNames, Fields: spawnTabDomainFields}),
	actArg("AttachDomain", "Attach a named domain", Field{Name: "AttachDomain", Kind: String}),
	actArg("CopyMode", "Copy-mode / search-mode sub-action", CopyModeUnion),
	actArg("RotatePanes", "Rotate the panes in the tab", Field{Name: "RotatePanes", Kind: Enum, Enum: []string{"Clockwise", "CounterClockwise"}}),
	actArg("SplitPane", "Split with size and command", Field{Name: "SplitPane", Kind: Struct, Fields: []Field{
		{Name: "direction", Kind: Enum, Enum: []string{"Up", "Down", "Left", "Right"}, Required: true},
		{Name: "size", Kind: Union, Enum: []string{"Cells", "Percent"}, Fields: []Field{
			{Name: "Cells", Kind: Union, Scalar: true, Fields: []Field{{Name: "Cells", Kind: Int, Min: ptr(1)}}},
			{Name: "Percent", Kind: Union, Scalar: true, Fields: []Field{{Name: "Percent", Kind: Int, Min: ptr(1), Max: ptr(100)}}},
		}},
		{Name: "command", Kind: Struct, Fields: SpawnCommandFields},
		{Name: "top_level", Kind: Bool},
	}}),
	actArg("PaneSelect", "Pick a pane by label", Field{Name: "PaneSelect", Kind: Struct, Fields: []Field{
		{Name: "alphabet", Kind: String},
		{Name: "mode", Kind: Enum, Enum: []string{"Activate", "SwapWithActive", "SwapWithActiveKeepFocus", "MoveToNewTab", "MoveToNewWindow"}},
		{Name: "show_pane_ids", Kind: Bool},
	}}),
	actArg("CharSelect", "Character/emoji picker", Field{Name: "CharSelect", Kind: Struct, Fields: []Field{
		{Name: "group", Kind: Enum, Enum: []string{"RecentlyUsed", "SmileysAndEmotion", "PeopleAndBody", "AnimalsAndNature", "FoodAndDrink", "TravelAndPlaces", "Activities", "Objects", "Symbols", "Flags", "NerdFonts", "UnicodeNames", "ShortCodes"}},
		{Name: "copy_on_select", Kind: Bool},
		{Name: "copy_to", Kind: Enum, Enum: []string{"Clipboard", "PrimarySelection", "ClipboardAndPrimarySelection"}},
	}}),
	act("ResetTerminal", "Reset the terminal (RIS)"),
	actArg("OpenUri", "Open a URI", Field{Name: "OpenUri", Kind: String}),
	act("ActivateCommandPalette", "Open the command palette"),
	actArg("ActivateWindow", "Activate the Nth window", Field{Name: "ActivateWindow", Kind: Int, Min: ptr(0)}),
	actArg("ActivateWindowRelative", "Activate window by offset, wrapping", Field{Name: "ActivateWindowRelative", Kind: Int}),
	actArg("ActivateWindowRelativeNoWrap", "Activate window by offset, no wrap", Field{Name: "ActivateWindowRelativeNoWrap", Kind: Int}),
	actArg("PromptInputLine", "Prompt for a line of input", Field{Name: "PromptInputLine", Kind: Struct, Fields: []Field{
		{Name: "action", Kind: Lua, Required: true, Default: "wezterm.action_callback(function(window, pane, line)\n  -- line is nil when cancelled\nend)"},
		{Name: "description", Kind: String},
		{Name: "prompt", Kind: String},
		{Name: "initial_value", Kind: String},
	}}),
	actArg("InputSelector", "Overlay list of choices", Field{Name: "InputSelector", Kind: Struct, Fields: []Field{
		{Name: "action", Kind: Lua, Required: true, Default: "wezterm.action_callback(function(window, pane, id, label)\n  -- your code here\nend)"},
		{Name: "title", Kind: String},
		{Name: "choices", Kind: List, Required: true, Fields: []Field{
			{Name: "label", Kind: String, Required: true},
			{Name: "id", Kind: String},
		}},
		{Name: "fuzzy", Kind: Bool},
		{Name: "alphabet", Kind: String},
		{Name: "description", Kind: String},
		{Name: "fuzzy_description", Kind: String},
	}}),
	actArg("Confirmation", "Yes/no confirmation", Field{Name: "Confirmation", Kind: Struct, Fields: []Field{
		{Name: "action", Kind: Lua, Required: true, Default: "wezterm.action_callback(function(window, pane)\n  -- your code here\nend)"},
		{Name: "cancel", Kind: Lua},
		{Name: "message", Kind: String},
	}}),
}

// Mark struct-arg actions whose arg is all-optional as BareOK (Appendix C).
func init() {
	for _, n := range []string{"SpawnTab", "SpawnCommandInNewTab", "SpawnCommandInNewWindow", "SplitHorizontal", "SplitVertical", "ClearScrollback", "QuickSelectArgs", "SwitchToWorkspace", "PaneSelect", "CharSelect"} {
		if a := FindAction(n); a != nil {
			a.BareOK = true
		}
	}
}

// Actions lists all 84 key/mouse action constructors in catalog order.
var Actions = actionDefs

// FindAction returns the action definition with the given name, or nil.
func FindAction(name string) *ActionDef {
	for i := range Actions {
		if Actions[i].Name == name {
			return &Actions[i]
		}
	}
	return nil
}
