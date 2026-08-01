package bridge

import "errors"

var ErrTranslatorUnavailable = errors.New("geyser-go: no Java-to-Bedrock translator is installed")
