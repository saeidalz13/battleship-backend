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

	rawGrid, ok := value.([]interface{})
	if !ok {
		return nil, cerr.ErrValueNotGridInt()
	}

	gridInt, err := convertToGridInt(rawGrid)
	if err != nil {
		return nil, err
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

// First it checks if every row of this slice of interface is a slice itself
// Second, it checks if every value of the slice is numeric (default float64 for json)
// If all the checks are ok, then we set the positions of out new GridInt
func convertToGridInt(rawGrid []interface{}) (md.GridInt, error) {
	gridInt := make(md.GridInt, len(rawGrid))

	for i, rawRow := range rawGrid {
		// is the row a slice or not
		row, ok := rawRow.([]interface{})
		if !ok {
			return nil, cerr.ErrValueNotGridInt()
		}

		gridRow := make([]int, len(row))
		for j, rawRowValue := range row {
			rowValue, ok := rawRowValue.(float64) // json defaults to float64
			if !ok {
				return nil, cerr.ErrValueNotGridInt()
			}

			gridRow[j] = int(rowValue)
		}

		gridInt[i] = gridRow
	}
	return gridInt, nil
}
