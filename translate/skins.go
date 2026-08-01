package translate

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	maxJavaSkinImageBytes = 8 << 20
	maxJavaSkinPixels     = 128 * 128
)

var javaSkinHTTPClient = &http.Client{Timeout: 5 * time.Second}

type javaTextureProperty struct {
	URL      string `json:"url"`
	Metadata struct {
		Model string `json:"model"`
	} `json:"metadata"`
}

type javaTexturesProperty struct {
	Textures struct {
		Skin javaTextureProperty `json:"SKIN"`
		Cape javaTextureProperty `json:"CAPE"`
	} `json:"textures"`
}

// javaTexturesPropertyValue returns the first Minecraft textures property.
// Java profile properties are server-supplied data, so malformed or repeated
// properties are handled as a missing skin rather than as a session error.
func javaTexturesPropertyValue(properties []javaprotocol.Property) (string, bool) {
	for _, property := range properties {
		if property.Name == "textures" && property.Value != "" {
			return property.Value, true
		}
	}
	return "", false
}

func decodeJavaTexturesProperty(value string) (javaTexturesProperty, error) {
	payload, err := decodeBase64(value)
	if err != nil {
		return javaTexturesProperty{}, fmt.Errorf("textures property base64: %w", err)
	}
	var textures javaTexturesProperty
	if err := json.Unmarshal(payload, &textures); err != nil {
		return javaTexturesProperty{}, fmt.Errorf("textures property JSON: %w", err)
	}
	return textures, nil
}

func decodeBase64(value string) ([]byte, error) {
	decoders := []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding}
	var lastErr error
	for _, decoder := range decoders {
		decoded, err := decoder.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func javaTextureURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" || strings.ToLower(parsed.Hostname()) != "textures.minecraft.net" || parsed.Port() != "" || parsed.User != nil {
		return nil, fmt.Errorf("texture URL is not an HTTPS textures.minecraft.net URL")
	}
	if parsed.Path == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("texture URL has an invalid path or suffix")
	}
	return parsed, nil
}

func downloadJavaTexture(raw string) (image.Image, error) {
	parsed, err := javaTextureURL(raw)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := javaSkinHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("texture server returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxJavaSkinImageBytes {
		return nil, fmt.Errorf("texture image is too large: %d bytes", response.ContentLength)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxJavaSkinImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxJavaSkinImageBytes {
		return nil, fmt.Errorf("texture image exceeds %d bytes", maxJavaSkinImageBytes)
	}
	decoded, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("decode texture image: %w", err)
	}
	return decoded, nil
}

func javaImageRGBA(source image.Image) ([]byte, uint32, uint32, error) {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || int64(width)*int64(height) > maxJavaSkinPixels {
		return nil, 0, 0, fmt.Errorf("texture dimensions %dx%d exceed limits", width, height)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), source, bounds.Min, draw.Src)
	return append([]byte(nil), rgba.Pix...), uint32(width), uint32(height), nil
}

func validJavaSkinDimensions(width, height uint32) bool {
	return (width == 64 && (height == 32 || height == 64)) || (width == 128 && height == 128)
}

func validJavaCapeDimensions(width, height uint32) bool {
	return width == 64 && height == 32
}

// projectJavaSkin resolves the Mojang textures property into the Bedrock skin
// structure used by PlayerList. It deliberately accepts only Mojang's texture
// host, bounds response size, and falls back to the caller's synthetic skin on
// malformed or unavailable remote data.
func projectJavaSkin(properties []javaprotocol.Property, name string) (gtprotocol.Skin, bool, error) {
	value, ok := javaTexturesPropertyValue(properties)
	if !ok {
		return gtprotocol.Skin{}, false, nil
	}
	textures, err := decodeJavaTexturesProperty(value)
	if err != nil {
		return gtprotocol.Skin{}, false, err
	}
	if textures.Textures.Skin.URL == "" {
		return gtprotocol.Skin{}, false, fmt.Errorf("textures property has no skin URL")
	}
	skinImage, err := downloadJavaTexture(textures.Textures.Skin.URL)
	if err != nil {
		return gtprotocol.Skin{}, false, err
	}
	skinData, width, height, err := javaImageRGBA(skinImage)
	if err != nil {
		return gtprotocol.Skin{}, false, err
	}
	if !validJavaSkinDimensions(width, height) {
		return gtprotocol.Skin{}, false, fmt.Errorf("unsupported Java skin dimensions %dx%d", width, height)
	}

	digest := sha256.Sum256([]byte(textures.Textures.Skin.URL))
	result := defaultPlayerSkin(name)
	result.SkinID = "java-" + hex.EncodeToString(digest[:16])
	result.SkinData = skinData
	result.SkinImageWidth = width
	result.SkinImageHeight = height
	result.Trusted = true
	if textures.Textures.Skin.Metadata.Model == "slim" {
		result.ArmSize = "slim"
	}
	if capeURL := textures.Textures.Cape.URL; capeURL != "" {
		if capeImage, capeErr := downloadJavaTexture(capeURL); capeErr == nil {
			if capeData, capeWidth, capeHeight, imageErr := javaImageRGBA(capeImage); imageErr == nil && validJavaCapeDimensions(capeWidth, capeHeight) {
				result.CapeData = capeData
				result.CapeImageWidth = capeWidth
				result.CapeImageHeight = capeHeight
				result.CapeID = "java-" + hex.EncodeToString(digest[:8])
			}
		}
	}
	return result, true, nil
}
