package api

import (
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

func TypeAssertStringPayload(initMap map[string]interface{}, keys ...string) ([]string, error) {
	desiredStrings := make([]string, len(keys))

	for i, key := range keys {
		value, prs := initMap[key]
		if !prs {
			return nil, cerr.ErrKeyNotExists(key)
		}

		str, ok := value.(string)
		if !ok {
			return nil, cerr.ErrValueNotString(value)
		}
		desiredStrings[i] = str
	}

	return desiredStrings, nil
}

func TypeAssertGridIntPayload(initMap map[string]interface{}, key string) (md.GridInt, error) {
	value, prs := initMap[key]
	if !prs {
		return nil, cerr.ErrKeyNotExists(key)
	}

	gridInt, ok := value.(md.GridInt)
	if !ok {
		return nil, cerr.ErrValueNotString(value)
	}
	return gridInt, nil
}

func TypeAssertPayloadToMap(payload interface{}) (map[string]interface{}, error) {
	initMap, ok := payload.(map[string]interface{})
	if !ok {
		return nil, cerr.ErrNilPayload()
	}
	return initMap, nil
}

func TypeAssertIntPayload(initMap map[string]interface{}, keys ...string) ([]int, error) {
	desiredInts := make([]int, len(keys))

	for i, key := range keys {
		value, prs := initMap[key]
		if !prs {
			return nil, cerr.ErrKeyNotExists(key)
		}

		intValue, ok := value.(int)
		if !ok {
			return nil, cerr.ErrValueNotInt(value)
		}

		desiredInts[i] = intValue
	}
	return desiredInts, nil
}
