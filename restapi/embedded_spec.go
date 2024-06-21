// Code generated by go-swagger; DO NOT EDIT.

package restapi

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
)

var (
	// SwaggerJSON embedded version of the swagger document used at generation time
	SwaggerJSON json.RawMessage
	// FlatSwaggerJSON embedded flattened version of the swagger document used at generation time
	FlatSwaggerJSON json.RawMessage
)

func init() {
	SwaggerJSON = json.RawMessage([]byte(`{
  "schemes": [
    "http"
  ],
  "swagger": "2.0",
  "info": {
    "description": "API for handling blob query in the Blob Hub.",
    "title": "Blob Hub Service API",
    "version": "1.0.0"
  },
  "host": "blob-hub",
  "basePath": "/eth/v1",
  "paths": {
    "/beacon/blob_sidecars/{block_id}": {
      "get": {
        "produces": [
          "application/json"
        ],
        "tags": [
          "blob"
        ],
        "summary": "Get blob sidecars by block num",
        "operationId": "getBlobSidecarsByBlockNum",
        "parameters": [
          {
            "minLength": 1,
            "type": "string",
            "description": "Block identifier. Can be one of: 'head' (canonical head in node's view), 'genesis', 'finalized', \u003cslot\u003e, \u003chex encoded blockRoot with 0x prefix\u003e",
            "name": "block_id",
            "in": "path",
            "required": true
          },
          {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "Array of indices for blob sidecars to request for in the specified block. Returns all blob sidecars in the block if not specified",
            "name": "indices",
            "in": "query"
          }
        ],
        "responses": {
          "200": {
            "description": "successful operation",
            "schema": {
              "$ref": "#/definitions/GetBlobSideCarsResponse"
            }
          },
          "400": {
            "description": "Bad Request",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          },
          "404": {
            "description": "blob not found",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          },
          "500": {
            "description": "internal server error",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Error": {
      "type": "object",
      "properties": {
        "code": {
          "description": "HTTP error code",
          "type": "integer",
          "format": "int64",
          "x-omitempty": false,
          "example": "400/500"
        },
        "message": {
          "description": "Error message",
          "type": "string",
          "x-omitempty": false,
          "example": "Bad request/Internal server error"
        }
      }
    },
    "GetBlobSideCarsResponse": {
      "type": "object",
      "properties": {
        "code": {
          "description": "status code",
          "type": "integer",
          "example": 200
        },
        "data": {
          "description": "actual data for request",
          "type": "array",
          "items": {
            "$ref": "#/definitions/Sidecar"
          }
        },
        "message": {
          "description": "error message if there is error",
          "type": "string",
          "example": "signature invalid"
        }
      }
    },
    "Sidecar": {
      "type": "object",
      "properties": {
        "blob": {
          "type": "string"
        },
        "index": {
          "type": "string",
          "example": 1
        },
        "kzg_commitment": {
          "type": "string"
        },
        "kzg_commitment_inclusion_proof": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "x-omitempty": true
        },
        "kzg_proof": {
          "type": "string"
        },
        "signed_block_header": {
          "type": "object",
          "properties": {
            "message": {
              "type": "object",
              "properties": {
                "body_root": {
                  "type": "string"
                },
                "parent_root": {
                  "type": "string"
                },
                "proposer_index": {
                  "type": "string"
                },
                "slot": {
                  "type": "string"
                },
                "state_root": {
                  "type": "string"
                }
              }
            },
            "signature": {
              "type": "string"
            }
          }
        }
      }
    }
  }
}`))
	FlatSwaggerJSON = json.RawMessage([]byte(`{
  "schemes": [
    "http"
  ],
  "swagger": "2.0",
  "info": {
    "description": "API for handling blob query in the Blob Hub.",
    "title": "Blob Hub Service API",
    "version": "1.0.0"
  },
  "host": "blob-hub",
  "basePath": "/eth/v1",
  "paths": {
    "/beacon/blob_sidecars/{block_id}": {
      "get": {
        "produces": [
          "application/json"
        ],
        "tags": [
          "blob"
        ],
        "summary": "Get blob sidecars by block num",
        "operationId": "getBlobSidecarsByBlockNum",
        "parameters": [
          {
            "minLength": 1,
            "type": "string",
            "description": "Block identifier. Can be one of: 'head' (canonical head in node's view), 'genesis', 'finalized', \u003cslot\u003e, \u003chex encoded blockRoot with 0x prefix\u003e",
            "name": "block_id",
            "in": "path",
            "required": true
          },
          {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "Array of indices for blob sidecars to request for in the specified block. Returns all blob sidecars in the block if not specified",
            "name": "indices",
            "in": "query"
          }
        ],
        "responses": {
          "200": {
            "description": "successful operation",
            "schema": {
              "$ref": "#/definitions/GetBlobSideCarsResponse"
            }
          },
          "400": {
            "description": "Bad Request",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          },
          "404": {
            "description": "blob not found",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          },
          "500": {
            "description": "internal server error",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Error": {
      "type": "object",
      "properties": {
        "code": {
          "description": "HTTP error code",
          "type": "integer",
          "format": "int64",
          "x-omitempty": false,
          "example": "400/500"
        },
        "message": {
          "description": "Error message",
          "type": "string",
          "x-omitempty": false,
          "example": "Bad request/Internal server error"
        }
      }
    },
    "GetBlobSideCarsResponse": {
      "type": "object",
      "properties": {
        "code": {
          "description": "status code",
          "type": "integer",
          "example": 200
        },
        "data": {
          "description": "actual data for request",
          "type": "array",
          "items": {
            "$ref": "#/definitions/Sidecar"
          }
        },
        "message": {
          "description": "error message if there is error",
          "type": "string",
          "example": "signature invalid"
        }
      }
    },
    "Sidecar": {
      "type": "object",
      "properties": {
        "blob": {
          "type": "string"
        },
        "index": {
          "type": "string",
          "example": 1
        },
        "kzg_commitment": {
          "type": "string"
        },
        "kzg_commitment_inclusion_proof": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "x-omitempty": true
        },
        "kzg_proof": {
          "type": "string"
        },
        "signed_block_header": {
          "type": "object",
          "properties": {
            "message": {
              "type": "object",
              "properties": {
                "body_root": {
                  "type": "string"
                },
                "parent_root": {
                  "type": "string"
                },
                "proposer_index": {
                  "type": "string"
                },
                "slot": {
                  "type": "string"
                },
                "state_root": {
                  "type": "string"
                }
              }
            },
            "signature": {
              "type": "string"
            }
          }
        }
      }
    },
    "SidecarSignedBlockHeader": {
      "type": "object",
      "properties": {
        "message": {
          "type": "object",
          "properties": {
            "body_root": {
              "type": "string"
            },
            "parent_root": {
              "type": "string"
            },
            "proposer_index": {
              "type": "string"
            },
            "slot": {
              "type": "string"
            },
            "state_root": {
              "type": "string"
            }
          }
        },
        "signature": {
          "type": "string"
        }
      }
    },
    "SidecarSignedBlockHeaderMessage": {
      "type": "object",
      "properties": {
        "body_root": {
          "type": "string"
        },
        "parent_root": {
          "type": "string"
        },
        "proposer_index": {
          "type": "string"
        },
        "slot": {
          "type": "string"
        },
        "state_root": {
          "type": "string"
        }
      }
    }
  }
}`))
}
