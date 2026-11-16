package theme

func builtInThemes() []Theme {
	return []Theme{
		{
			ID: "sharp", Name: "Sharp",
			Light: Palette{
				Text: "#283445", Muted: "#667386", Border: "#bac3d0", Focus: "#416e28", Accent: "#325d99",
				Category: "#283445", Group: "#667386", Action: "#283445", Shortcut: "#325d99",
				SelectionForeground: "#283445", SelectionBackground: "#d4e1f5",
				RetainedForeground: "#283445", RetainedBackground: "#e8edf4",
				Success: "#416e28", Warning: "#916413", Error: "#b43b46",
			},
			Dark: Palette{
				Text: "#d5dbe7", Muted: "#8b96ad", Border: "#454b62", Focus: "#a6ce72", Accent: "#b4caff",
				Category: "#d5dbe7", Group: "#8b96ad", Action: "#d5dbe7", Shortcut: "#b4caff",
				SelectionForeground: "#d5dbe7", SelectionBackground: "#3d5485",
				RetainedForeground: "#d5dbe7", RetainedBackground: "#272e42",
				Success: "#a6ce72", Warning: "#e5be78", Error: "#e68b92",
			},
		},
		{
			ID: "sharp-color", Name: "Sharp Color",
			Light: Palette{
				Text: "#243042", Muted: "#657386", Border: "#aeb9c8", Focus: "#28785c", Accent: "#2f63a3",
				Category: "#236c85", Group: "#7352a3", Action: "#9a4f20", Shortcut: "#2f63a3",
				SelectionForeground: "#172033", SelectionBackground: "#c9ddf5",
				RetainedForeground: "#314f7a", RetainedBackground: "#e5edf7",
				Success: "#28785c", Warning: "#9a680d", Error: "#b23a48",
			},
			Dark: Palette{
				Text: "#d8dee9", Muted: "#8f9bb2", Border: "#495269", Focus: "#8bd5a6", Accent: "#82aaff",
				Category: "#73c7ec", Group: "#c5a3ff", Action: "#f2a86f", Shortcut: "#82aaff",
				SelectionForeground: "#f4f7fb", SelectionBackground: "#31456d",
				RetainedForeground: "#b8cbff", RetainedBackground: "#273247",
				Success: "#8bd5a6", Warning: "#edc879", Error: "#f08b98",
			},
		},
		{
			ID: "lazygit", Name: "Lazygit",
			Light: Palette{
				Text: "#20242d", Muted: "#626b7a", Border: "#9ba4b4", Focus: "#007c7a", Accent: "#005faf",
				Category: "#005faf", Group: "#7b3fa1", Action: "#a84a00", Shortcut: "#007c7a",
				SelectionForeground: "#ffffff", SelectionBackground: "#006f8b",
				RetainedForeground: "#21395c", RetainedBackground: "#d6e9f2",
				Success: "#267326", Warning: "#8a6500", Error: "#b42335",
			},
			Dark: Palette{
				Text: "#e6edf3", Muted: "#8b949e", Border: "#484f58", Focus: "#2dd4bf", Accent: "#58a6ff",
				Category: "#58a6ff", Group: "#d2a8ff", Action: "#ffa657", Shortcut: "#39c5cf",
				SelectionForeground: "#ffffff", SelectionBackground: "#0d6478",
				RetainedForeground: "#a5d6ff", RetainedBackground: "#26394b",
				Success: "#56d364", Warning: "#e3b341", Error: "#ff7b72",
			},
		},
		{
			ID: "catppuccin", Name: "Catppuccin",
			Light: Palette{
				Text: "#4c4f69", Muted: "#7c7f93", Border: "#bcc0cc", Focus: "#179299", Accent: "#1e66f5",
				Category: "#1e66f5", Group: "#8839ef", Action: "#fe640b", Shortcut: "#179299",
				SelectionForeground: "#4c4f69", SelectionBackground: "#ccd0da",
				RetainedForeground: "#1e66f5", RetainedBackground: "#e6e9ef",
				Success: "#40a02b", Warning: "#df8e1d", Error: "#d20f39",
			},
			Dark: Palette{
				Text: "#cdd6f4", Muted: "#9399b2", Border: "#585b70", Focus: "#94e2d5", Accent: "#89b4fa",
				Category: "#89b4fa", Group: "#cba6f7", Action: "#fab387", Shortcut: "#94e2d5",
				SelectionForeground: "#1e1e2e", SelectionBackground: "#89b4fa",
				RetainedForeground: "#b4befe", RetainedBackground: "#313244",
				Success: "#a6e3a1", Warning: "#f9e2af", Error: "#f38ba8",
			},
		},
		{
			ID: "high-contrast", Name: "High Contrast",
			Light: Palette{
				Text: "#000000", Muted: "#454545", Border: "#5c5c5c", Focus: "#005a30", Accent: "#003f9e",
				Category: "#003f9e", Group: "#6b238e", Action: "#8f3b00", Shortcut: "#005a6b",
				SelectionForeground: "#ffffff", SelectionBackground: "#003f9e",
				RetainedForeground: "#000000", RetainedBackground: "#c8dcff",
				Success: "#006400", Warning: "#6f5200", Error: "#a00020",
			},
			Dark: Palette{
				Text: "#ffffff", Muted: "#c8c8c8", Border: "#a0a0a0", Focus: "#64ffb4", Accent: "#73b7ff",
				Category: "#73b7ff", Group: "#e3a6ff", Action: "#ffc078", Shortcut: "#62f3ff",
				SelectionForeground: "#000000", SelectionBackground: "#fff06a",
				RetainedForeground: "#ffffff", RetainedBackground: "#3b4d70",
				Success: "#72ff72", Warning: "#ffe66b", Error: "#ff707f",
			},
		},
	}
}
