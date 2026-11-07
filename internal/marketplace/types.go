package marketplace

import "encoding/json"

type Image struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Tag  string `json:"tag"`
}
type Amount struct {
	Currency string `json:"currency"`
	Label    string `json:"label,omitempty"`
	Value    int64  `json:"value"`
}
type Price struct {
	Amounts  []Amount `json:"amounts"`
	Units    int64    `json:"units"`
	Duration int64    `json:"duration"`
}
type Reference struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
type Provenance struct {
	Source      string `json:"source"`
	Relation    string `json:"relation"`
	ID          string `json:"id"`
	Description string `json:"description"`
}
type Item struct {
	Preview       *ModelPreview   `json:"preview,omitempty"`
	ID            string          `json:"id"`
	TitleID       string          `json:"titleId"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Creator       string          `json:"creator"`
	CreatorID     string          `json:"creatorId"`
	Type          string          `json:"type"`
	Kind          string          `json:"kind"`
	Images        []Image         `json:"images"`
	Tags          []string        `json:"tags"`
	Platforms     []string        `json:"platforms"`
	IDs           []string        `json:"ids"`
	Prices        []Price         `json:"prices"`
	Free          bool            `json:"free"`
	Sale          bool            `json:"sale"`
	Rating        *float64        `json:"rating"`
	Ratings       int64           `json:"ratings"`
	MinVersion    string          `json:"minVersion"`
	MaxVersion    string          `json:"maxVersion"`
	Compatibility string          `json:"compatibility"`
	Start         string          `json:"start"`
	End           string          `json:"end"`
	Modified      string          `json:"modified"`
	References    []Reference     `json:"references"`
	Link          string          `json:"link"`
	Technical     json.RawMessage `json:"technical"`
	Provenance    []Provenance    `json:"provenance"`
}
type ModelPreview struct {
	Geometry json.RawMessage `json:"geometry"`
	Texture  string          `json:"texture"`
}
type Source struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	ExperienceID string `json:"experienceId"`
	TitleID      string `json:"titleId"`
	StorePageID  string `json:"storePageId"`
	CreatorID    string `json:"creatorId"`
	Creator      string `json:"creator"`
	Overview     Item   `json:"overview"`
}
type Discovery struct {
	Revision  string   `json:"revision"`
	Sources   []Source `json:"sources"`
	CheckedAt string   `json:"checkedAt"`
}
type Query struct {
	Creator     string `json:"creator"`
	Environment string `json:"environment"`
	Source      string `json:"source"`
	Relation    string `json:"relation"`
	Search      string `json:"search"`
	Cursor      string `json:"cursor"`
	Count       int    `json:"count"`
	Sort        string `json:"sort"`
	Type        string `json:"type"`
	Tag         string `json:"tag"`
	Price       string `json:"price"`
	Currency    string `json:"currency"`
	MaxPrice    *int64 `json:"maxPrice"`
	Compatible  bool   `json:"compatible"`
}
type Page struct {
	Items     []Item   `json:"items"`
	Cursor    string   `json:"cursor"`
	Warnings  []string `json:"warnings"`
	CheckedAt string   `json:"checkedAt"`
	Relation  string   `json:"relation"`
}
type Favorite struct {
	Environment string `json:"environment"`
	Source      string `json:"source"`
	Item        string `json:"item"`
	Relation    string `json:"relation"`
}
