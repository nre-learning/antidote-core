package swagger 

const (
Lessondef = `{
  "swagger": "2.0",
  "info": {
    "title": "api/exp/definitions/lessondef.proto",
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
      "post": {
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
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/expLessonDefFilter"
            }
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
    }
  },
  "definitions": {
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
        }
      }
    },
    "expLessonDefFilter": {
      "type": "object",
      "properties": {
        "Category": {
          "type": "string"
        }
      }
    },
    "expLessonDefs": {
      "type": "object",
      "properties": {
        "lessondefs": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLessonDef"
          }
        },
        "Category": {
          "type": "string"
        }
      }
    },
    "expLessonStage": {
      "type": "object",
      "properties": {
        "StageId": {
          "type": "integer",
          "format": "int32"
        },
        "Description": {
          "type": "string"
        }
      }
    }
  }
}
`
Livelesson = `{
  "swagger": "2.0",
  "info": {
    "title": "api/exp/definitions/livelesson.proto",
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
    "/exp/livelesson/all": {
      "get": {
        "summary": "Retrieve all livelessons",
        "operationId": "ListLiveLessons",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expLiveLessons"
            }
          }
        },
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
    "EndpointEndpointType": {
      "type": "string",
      "enum": [
        "UNKNOWN",
        "DEVICE",
        "NOTEBOOK",
        "BLACKBOX",
        "UTILITY"
      ],
      "default": "UNKNOWN",
      "description": "This field helps the web client understand how to connect to this endpoint. Some might be done via SSH/Guacamole, others might be iframes, etc."
    },
    "expEndpoint": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Type": {
          "$ref": "#/definitions/EndpointEndpointType"
        },
        "Host": {
          "type": "string",
          "description": "This will contain a ClusterIP for SSH endpoints, so we don't need to allocate a public IP for them. If a NOTEBOOK,\nthis will get set to the FQDN needed to connect to the external IP allocated for it."
        },
        "Port": {
          "type": "integer",
          "format": "int32",
          "description": "Port for normal operations. If type \"device\", used for SSH. If type \"notebook\", loads up in an iframe."
        },
        "Protocol": {
          "type": "string",
          "title": "Future stuff for NOTEBOOK type"
        },
        "Uri": {
          "type": "string"
        },
        "Api_port": {
          "type": "integer",
          "format": "int32",
          "description": "Extra port for API interactions.\nNote that this mostly doesn't matter, as this is only in case we want to allow API interactions from outside\nthe cluster via NodePort. Most of the time we'll use the standard ports using in-cluster traffic."
        }
      }
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
        "Endpoints": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expEndpoint"
          }
        },
        "LessonStage": {
          "type": "integer",
          "format": "int32"
        },
        "LabGuide": {
          "type": "string"
        },
        "Ready": {
          "type": "boolean",
          "format": "boolean"
        }
      },
      "description": "A provisioned lab without the scheduler details. The server will translate from an underlying type\n(i.e. KubeLab) into this, so only the abstract, relevant details are presented."
    },
    "expLiveLessons": {
      "type": "object",
      "properties": {
        "livelessons": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLiveLessons"
          }
        }
      }
    }
  }
}
`
)
