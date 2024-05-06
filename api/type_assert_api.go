package api

import (
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

func TypeAssertStringPayload(payload interface{}, keys ...string) (map[string]string, error) {
	finalMap := make(map[string]string)
	desiredMap, ok := payload.(map[string]interface{})
	if !ok {
		return nil, cerr.ErrorNilPayload()
	}

	for _, key := range keys {
		value, prs := desiredMap[key]
		if !prs {
			return nil, cerr.ErrKeyNotExists(key)
		}

		str, ok := value.(string)
		if !ok {
			return nil, cerr.ErrValueNotString(value)
		}
		finalMap[key] = str
	}
	return finalMap, nil
}

func TypeAssertGridIntPayload(payload interface{}, key string) (md.GridInt, error) {
	desiredMap, ok := payload.(map[string]interface{})
	if !ok {
		return nil, cerr.ErrorNilPayload()
	}

	value, prs := desiredMap[key]
	if !prs {
		return nil, cerr.ErrKeyNotExists(key)
	}

	gridInt, ok := value.(md.GridInt)
	if !ok {
		return nil, cerr.ErrValueNotString(value)
	}
	return gridInt, nil
}
