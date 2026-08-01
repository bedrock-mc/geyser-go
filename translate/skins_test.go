package translate

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJavaTexturesProperty(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"timestamp": 1,
		"profileId": "00000000000000000000000000000000",
		"textures": map[string]any{
			"SKIN": map[string]any{
				"url":      "https://textures.minecraft.net/texture/skin",
				"metadata": map[string]string{"model": "slim"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	value := base64.StdEncoding.EncodeToString(payload)
	textures, err := decodeJavaTexturesProperty(value)
	if err != nil {
		t.Fatal(err)
	}
	if textures.Textures.Skin.URL != "https://textures.minecraft.net/texture/skin" || textures.Textures.Skin.Metadata.Model != "slim" {
		t.Fatalf("decoded Java textures = %#v", textures)
	}
}

func TestJavaTextureURLIsRestrictedToMojang(t *testing.T) {
	if _, err := javaTextureURL("https://textures.minecraft.net/texture/abc"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"http://textures.minecraft.net/texture/abc",
		"https://example.invalid/texture/abc",
		"https://textures.minecraft.net:443/texture/abc",
		"https://textures.minecraft.net/texture/abc?x=1",
	} {
		if _, err := javaTextureURL(value); err == nil {
			t.Fatalf("texture URL %q was accepted", value)
		}
	}
}

func TestProjectJavaSkinConvertsMojangTexture(t *testing.T) {
	imageData := make([]byte, 0)
	imageBuffer := new(bytes.Buffer)
	texture := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			texture.SetRGBA(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	if err := png.Encode(imageBuffer, texture); err != nil {
		t.Fatal(err)
	}
	imageData = imageBuffer.Bytes()

	previousClient := javaSkinHTTPClient
	javaSkinHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(bytes.NewReader(imageData)),
			ContentLength: int64(len(imageData)),
			Header:        make(http.Header),
			Request:       request,
		}, nil
	})}
	defer func() { javaSkinHTTPClient = previousClient }()

	propertyPayload := base64.StdEncoding.EncodeToString([]byte(`{"textures":{"SKIN":{"url":"https://textures.minecraft.net/texture/abc","metadata":{"model":"slim"}}}}`))
	skin, ok, err := projectJavaSkin([]javaprotocol.Property{{Name: "textures", Value: propertyPayload}}, "Player")
	if err != nil || !ok {
		t.Fatalf("projectJavaSkin = skin=%#v ok=%t err=%v", skin, ok, err)
	}
	if skin.SkinImageWidth != 64 || skin.SkinImageHeight != 64 || len(skin.SkinData) != 64*64*4 {
		t.Fatalf("skin dimensions/data = %dx%d/%d", skin.SkinImageWidth, skin.SkinImageHeight, len(skin.SkinData))
	}
	if skin.ArmSize != "slim" || skin.SkinData[0:4] == nil || skin.SkinData[0] != 10 || skin.SkinData[1] != 20 || skin.SkinData[2] != 30 || skin.SkinData[3] != 255 {
		t.Fatalf("skin projection = %#v", skin)
	}
}

func TestJavaImageRGBARejectsOversizedTexture(t *testing.T) {
	texture := image.NewRGBA(image.Rect(0, 0, 129, 128))
	if _, _, _, err := javaImageRGBA(texture); err == nil {
		t.Fatal("oversized texture unexpectedly converted")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
