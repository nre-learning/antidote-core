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
    },
    "/exp/livelessonall": {
      "get": {
        "summary": "Retrieve all livelessons",
        "operationId": "ListLiveLessons",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/expLiveLessonMap"
            }
          }
        },
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
        "IFRAME",
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
          "description": "This will contain a ClusterIP for SSH endpoints, so we don't need to allocate a public IP for them. If an IFRAME,\nthis will get set to the FQDN needed to connect to the external IP allocated for it."
        },
        "Port": {
          "type": "integer",
          "format": "int32"
        },
        "IframeDetails": {
          "$ref": "#/definitions/expIFDetails"
        },
        "Sshuser": {
          "type": "string"
        },
        "Sshpassword": {
          "type": "string"
        }
      }
    },
    "expHealthCheckMessage": {
      "type": "object"
    },
    "expIFDetails": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string"
        },
        "Protocol": {
          "type": "string"
        },
        "URI": {
          "type": "string"
        },
        "Port": {
          "type": "integer",
          "format": "int32"
        }
      }
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
    "expLessontoUUIDMap": {
      "type": "object",
      "properties": {
        "Uuids": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expUUIDtoLiveLessonMap"
          }
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
        },
        "createdTime": {
          "type": "string",
          "format": "date-time"
        },
        "sessionId": {
          "type": "string"
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
    "expLiveLessonMap": {
      "type": "object",
      "properties": {
        "Sessions": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLessontoUUIDMap"
          }
        }
      }
    },
    "expUUIDtoLiveLessonMap": {
      "type": "object",
      "properties": {
        "Livelessons": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveLesson"
          }
        }
      }
    }
  }
}
`
Syringeinfo = `{
  "swagger": "2.0",
  "info": {
    "title": "api/exp/definitions/syringeinfo.proto",
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
