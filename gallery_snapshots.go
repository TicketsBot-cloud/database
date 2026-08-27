package database

type GalleryPanelSnapshot struct {
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	Colour         int32   `json:"colour"`
	ImageUrl       *string `json:"image_url,omitempty"`
	ThumbnailUrl   *string `json:"thumbnail_url,omitempty"`
	ButtonStyle    *int16  `json:"button_style"`
	ButtonLabel    string  `json:"button_label"`
	EmojiName      *string `json:"emoji_name,omitempty"`
	WelcomeMessage []byte  `json:"welcome_message,omitempty"`
}

type GalleryTagSnapshot struct {
	Content *string              `json:"content,omitempty"`
	Embed   *CustomEmbedWithFields `json:"embed,omitempty"`
}

type GalleryFormSnapshot struct {
	Title  string                     `json:"title"`
	Inputs []GalleryFormInputSnapshot `json:"inputs"`
}

type GalleryFormInputSnapshot struct {
	Type        int                              `json:"type"`
	Position    int                              `json:"position"`
	Style       uint8                            `json:"style"`
	Label       string                           `json:"label"`
	Description *string                          `json:"description,omitempty"`
	Placeholder *string                          `json:"placeholder,omitempty"`
	Required    bool                             `json:"required"`
	MinLength   *uint16                          `json:"min_length,omitempty"`
	MaxLength   *uint16                          `json:"max_length,omitempty"`
	Options     []GalleryFormInputOptionSnapshot `json:"options,omitempty"`
}

type GalleryFormInputOptionSnapshot struct {
	Position    int     `json:"position"`
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
	Value       string  `json:"value"`
}
