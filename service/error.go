package service

import (
	"fmt"
)

// Verify Interface Compliance
var _ error = (*Err)(nil)

// Err defines service errors.
type Err struct {
	Code    int64  `json:"code"`
	Message string `json:"error"`
}

func (e Err) Enrich(message string) Err {
	return Err{
		Code:    e.Code,
		Message: fmt.Sprintf("%s: %s", e.Message, message),
	}
}

func (e Err) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

var (
	// NoErr - success
	NoErr = Err{Code: 2000}

	//CollectionFailCreateErr - fail to create collection
	CollectionFailCreateErr    = Err{3101, "fail to create collection"}
	CollectionFailGetErr       = Err{3102, "fail to get collection"}
	CollectionNotExistErr      = Err{3103, "collection not exist"}
	CollectionAlreadyExistErr  = Err{3104, "collection already exist"}
	CollectionInvalidTimestamp = Err{3105, "collection invalid timestamp"}
	CollectionInvalidCategory  = Err{3106, "invalid category"}
	CollectionMismatch         = Err{3107, "input does not match Greenfield data"}

	//DataFailCreateErr - fail to create data
	DataFailCreateErr    = Err{3201, "fail to create data"} // Not used now, too general
	DataFailGetErr       = Err{3202, "fail to get data"}    // Not used now, too general
	DataNotExistErr      = Err{3203, "data not exist"}
	DataAlreadyExistErr  = Err{3204, "data already exist"}
	DataInvalidTimestamp = Err{3205, "nft tx expired/invalid timestamp"}

	MediaInvalidFile     = Err{3401, "invalid media file"}
	MediaFailToSave      = Err{3402, "cannot save media"}
	MediaUnsupportedType = Err{3403, "unsupported media type"}
	MediaNotExist        = Err{3404, "media not exists"}

	//NotFoundErr - not found error
	NotFoundErr = Err{Code: 4000, Message: "cannot find"}

	//InternalErr -internal error
	InternalErr = Err{Code: 5000, Message: "internal error"}
)
