package swagger 

const (
Lessondef = `{
  "swagger": "2.0",
  "info": {
    "title": "lessondef.proto",
    "version": "version not set"
  },
  "schemes": [
    "http",
    "https"
  ],
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/exp/lessondef": {
      "get": {
        "summary": "Retrieve all LessonDefs with filter",
        "operationId": "ListLessonDefs",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expLessonDefs"
            }
          }
        },
        "parameters": [
          {
            "name": "Category",
            "in": "query",
            "required": false,
            "type": "string"
          }
        ],
        "tags": [
          "LessonDefService"
        ]
      }
    },
    "/exp/lessondef/{id}": {
      "get": {
        "operationId": "GetLessonDef",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expLessonDef"
            }
          }
        },
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "integer",
            "format": "int32"
          }
        ],
        "tags": [
          "LessonDefService"
        ]
      }
    },
    "/exp/lessondef/{id}/prereqs": {
      "get": {
        "operationId": "GetAllLessonPrereqs",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expLessonPrereqs"
            }
          }
        },
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "integer",
            "format": "int32"
          }
        ],
        "tags": [
          "LessonDefService"
        ]
      }
    }
  },
  "definitions": {
    "expBlackbox": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Image": {
          "type": "string"
        },
        "Ports": {
          "type": "array",
          "items": {
            "type": "integer",
            "format": "int32"
          }
        }
      }
    },
    "expConnection": {
      "type": "object",
      "properties": {
        "A": {
          "type": "string"
        },
        "B": {
          "type": "string"
        },
        "Subnet": {
          "type": "string"
        }
      }
    },
    "expDevice": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Image": {
          "type": "string"
        },
        "Ports": {
          "type": "array",
          "items": {
            "type": "integer",
            "format": "int32"
          }
        }
      }
    },
    "expIframeResource": {
      "type": "object",
      "properties": {
        "Ref": {
          "type": "string"
        },
        "Protocol": {
          "type": "string"
        },
        "Path": {
          "type": "string"
        },
        "Port": {
          "type": "integer",
          "format": "int32"
        }
      }
    },
    "expLessonDef": {
      "type": "object",
      "properties": {
        "LessonId": {
          "type": "integer",
          "format": "int32"
        },
        "Stages": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLessonStage"
          }
        },
        "LessonName": {
          "type": "string"
        },
        "IframeResources": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expIframeResource"
          }
        },
        "Devices": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expDevice"
          }
        },
        "Utilities": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expUtility"
          }
        },
        "Blackboxes": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expBlackbox"
          }
        },
        "Connections": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expConnection"
          }
        },
        "Category": {
          "type": "string"
        },
        "LessonDiagram": {
          "type": "string"
        },
        "LessonVideo": {
          "type": "string"
        },
        "Tier": {
          "type": "string"
        },
        "Prereqs": {
          "type": "array",
          "items": {
            "type": "integer",
            "format": "int32"
          }
        },
        "Tags": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "Collection": {
          "type": "string"
        },
        "Description": {
          "type": "string"
        },
        "Slug": {
          "type": "string",
          "title": "This is meant to fill: \"How well do you know \u003cslug\u003e?\""
        }
      }
    },
    "expLessonDefs": {
      "type": "object",
      "properties": {
        "lessonDefs": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLessonDef"
          }
        }
      }
    },
    "expLessonPrereqs": {
      "type": "object",
      "properties": {
        "prereqs": {
          "type": "array",
          "items": {
            "type": "integer",
            "format": "int32"
          }
        }
      }
    },
    "expLessonStage": {
      "type": "object",
      "properties": {
        "Id": {
          "type": "integer",
          "format": "int32"
        },
        "Description": {
          "type": "string"
        },
        "LabGuide": {
          "type": "string"
        },
        "JupyterLabGuide": {
          "type": "boolean",
          "format": "boolean"
        }
      }
    },
    "expUtility": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Image": {
          "type": "string"
        },
        "Ports": {
          "type": "array",
          "items": {
            "type": "integer",
            "format": "int32"
          }
        }
      }
    }
  }
}
`
Livelesson = `{
  "swagger": "2.0",
  "info": {
    "title": "livelesson.proto",
    "version": "version not set"
  },
  "schemes": [
    "http",
    "https"
  ],
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/*": {
      "get": {
        "operationId": "HealthCheck",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expHealthCheckMessage"
            }
          }
        },
        "tags": [
          "LiveLessonsService"
        ]
      }
    },
    "/exp/livelesson": {
      "post": {
        "summary": "Request a lab is created, or request the UUID of one that already exists for these parameters.",
        "operationId": "RequestLiveLesson",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expLessonUUID"
            }
          }
        },
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/expLessonParams"
            }
          }
        ],
        "tags": [
          "LiveLessonsService"
        ]
      }
    },
    "/exp/livelesson/{id}": {
      "get": {
        "summary": "Retrieve details about a lesson",
        "operationId": "GetLiveLesson",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expLiveLesson"
            }
          }
        },
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "tags": [
          "LiveLessonsService"
        ]
      }
    }
  },
  "definitions": {
    "LiveEndpointEndpointType": {
      "type": "string",
      "enum": [
        "UNKNOWN",
        "DEVICE",
        "IFRAME",
        "BLACKBOX",
        "UTILITY"
      ],
      "default": "UNKNOWN",
      "description": "This field helps the web client understand how to connect to this endpoint. Some might be done via SSH/Guacamole, others might be iframes, etc."
    },
    "expHealthCheckMessage": {
      "type": "object"
    },
    "expLessonParams": {
      "type": "object",
      "properties": {
        "lessonId": {
          "type": "integer",
          "format": "int32"
        },
        "sessionId": {
          "type": "string"
        },
        "lessonStage": {
          "type": "integer",
          "format": "int32"
        }
      }
    },
    "expLessonUUID": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string"
        }
      }
    },
    "expLiveEndpoint": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Type": {
          "$ref": "#/definitions/LiveEndpointEndpointType"
        },
        "Host": {
          "type": "string",
          "description": "This will contain a ClusterIP for SSH endpoints, so we don't need to allocate a public IP for them. If an IFRAME,\nthis will get set to the FQDN needed to connect to the external IP allocated for it."
        },
        "Port": {
          "type": "integer",
          "format": "int32"
        },
        "IframePath": {
          "type": "string"
        },
        "Reachable": {
          "type": "boolean",
          "format": "boolean"
        }
      }
    },
    "expLiveLesson": {
      "type": "object",
      "properties": {
        "LessonUUID": {
          "type": "string"
        },
        "LessonId": {
          "type": "integer",
          "format": "int32"
        },
        "LiveEndpoints": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveEndpoint"
          }
        },
        "LessonStage": {
          "type": "integer",
          "format": "int32"
        },
        "LabGuide": {
          "type": "string"
        },
        "JupyterLabGuide": {
          "type": "boolean",
          "format": "boolean"
        },
        "LiveLessonStatus": {
          "$ref": "#/definitions/expStatus"
        },
        "createdTime": {
          "type": "string",
          "format": "date-time"
        },
        "LessonDiagram": {
          "type": "string"
        },
        "LessonVideo": {
          "type": "string"
        },
        "Error": {
          "type": "boolean",
          "format": "boolean"
        }
      },
      "description": "A provisioned lab without the scheduler details. The server will translate from an underlying type\n(i.e. KubeLab) into this, so only the abstract, relevant details are presented."
    },
    "expSession": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string"
        }
      }
    },
    "expSessions": {
      "type": "object",
      "properties": {
        "sessions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expSession"
          }
        }
      }
    },
    "expStatus": {
      "type": "string",
      "enum": [
        "DONOTUSE",
        "INITIAL_BOOT",
        "CONFIGURATION",
        "READY"
      ],
      "default": "DONOTUSE"
    },
    "expSyringeState": {
      "type": "object",
      "properties": {
        "Livelessons": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveLesson"
          },
          "title": "Map that contains a mapping of UUIDs to LiveLesson messages"
        }
      }
    }
  }
}
`
Syringeinfo = `{
  "swagger": "2.0",
  "info": {
    "title": "syringeinfo.proto",
    "version": "version not set"
  },
  "schemes": [
    "http",
    "https"
  ],
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/exp/syringeinfo": {
      "get": {
        "operationId": "GetSyringeInfo",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expSyringeInfo"
            }
          }
        },
        "tags": [
          "SyringeInfoService"
        ]
      }
    }
  },
  "definitions": {
    "expSyringeInfo": {
      "type": "object",
      "properties": {
        "buildSha": {
          "type": "string"
        },
        "antidoteSha": {
          "type": "string"
        }
      }
    }
  }
}
`
)
