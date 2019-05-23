package swagger 

const (
Collection = `{
  "swagger": "2.0",
  "info": {
    "title": "collection.proto",
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
    "/exp/collection": {
      "get": {
        "operationId": "ListCollections",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expCollections"
            }
          }
        },
        "tags": [
          "CollectionService"
        ]
      }
    },
    "/exp/collection/{id}": {
      "get": {
        "operationId": "GetCollection",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expCollection"
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
          "CollectionService"
        ]
      }
    }
  },
  "definitions": {
    "expCollection": {
      "type": "object",
      "properties": {
        "Id": {
          "type": "integer",
          "format": "int32"
        },
        "Title": {
          "type": "string"
        },
        "Image": {
          "type": "string"
        },
        "Website": {
          "type": "string"
        },
        "ContactEmail": {
          "type": "string"
        },
        "BriefDescription": {
          "type": "string",
          "title": "Why should users view your collection?"
        },
        "LongDescription": {
          "type": "string",
          "title": "Why should users continue and view your lessons?"
        },
        "Type": {
          "type": "string"
        },
        "Tier": {
          "type": "string"
        },
        "CollectionFile": {
          "type": "string"
        },
        "Lessons": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLessonSummary"
          }
        }
      }
    },
    "expCollections": {
      "type": "object",
      "properties": {
        "collections": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expCollection"
          }
        }
      }
    },
    "expLessonSummary": {
      "type": "object",
      "properties": {
        "lessonId": {
          "type": "integer",
          "format": "int32"
        },
        "lessonName": {
          "type": "string"
        },
        "lessonDescription": {
          "type": "string"
        }
      }
    }
  }
}
`
Curriculum = `{
  "swagger": "2.0",
  "info": {
    "title": "curriculum.proto",
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
    "/exp/curriculum": {
      "get": {
        "operationId": "GetCurriculumInfo",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expCurriculumInfo"
            }
          }
        },
        "tags": [
          "CurriculumService"
        ]
      }
    }
  },
  "definitions": {
    "expCurriculumInfo": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Description": {
          "type": "string"
        },
        "Website": {
          "type": "string"
        }
      },
      "description": "Use this to return only metadata about the installed curriculum."
    }
  }
}
`
Kubelab = `{
  "swagger": "2.0",
  "info": {
    "title": "kubelab.proto",
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
  "paths": {},
  "definitions": {
    "expConnection": {
      "type": "object",
      "properties": {
        "A": {
          "type": "string"
        },
        "B": {
          "type": "string"
        }
      }
    },
    "expEndpoint": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Image": {
          "type": "string"
        },
        "configurationType": {
          "type": "string"
        },
        "Ports": {
          "type": "array",
          "items": {
            "type": "integer",
            "format": "int32"
          }
        },
        "Presentations": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expPresentation"
          }
        }
      }
    },
    "expKubeLab": {
      "type": "object",
      "properties": {
        "Namespace": {
          "type": "string"
        },
        "CreateRequest": {
          "$ref": "#/definitions/expLessonScheduleRequest"
        },
        "Networks": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "Pods": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "Services": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "Ingresses": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "status": {
          "$ref": "#/definitions/expStatus"
        },
        "ReachableEndpoints": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "CurrentStage": {
          "type": "integer",
          "format": "int32"
        }
      }
    },
    "expKubeLabs": {
      "type": "object",
      "properties": {
        "Items": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expKubeLab"
          }
        }
      }
    },
    "expLesson": {
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
        "Endpoints": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expEndpoint"
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
          "type": "integer",
          "format": "int32"
        },
        "Description": {
          "type": "string"
        },
        "Slug": {
          "type": "string",
          "title": "This is meant to fill: \"How well do you know \u003cslug\u003e?\""
        },
        "LessonFile": {
          "type": "string"
        },
        "LessonDir": {
          "type": "string"
        }
      }
    },
    "expLessonScheduleRequest": {
      "type": "object",
      "properties": {
        "Lesson": {
          "$ref": "#/definitions/expLesson"
        },
        "OperationType": {
          "type": "integer",
          "format": "int32"
        },
        "Uuid": {
          "type": "string"
        },
        "Stage": {
          "type": "integer",
          "format": "int32"
        },
        "Created": {
          "type": "string",
          "format": "date-time"
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
        },
        "VerifyCompleteness": {
          "type": "boolean",
          "format": "boolean"
        },
        "VerifyObjective": {
          "type": "string"
        }
      }
    },
    "expPresentation": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Port": {
          "type": "integer",
          "format": "int32"
        },
        "Type": {
          "type": "string"
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
    }
  }
}
`
Lesson = `{
  "swagger": "2.0",
  "info": {
    "title": "lesson.proto",
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
    "/exp/lesson": {
      "get": {
        "summary": "Retrieve all Lessons with filter",
        "operationId": "ListLessons",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expLessons"
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
          "LessonService"
        ]
      }
    },
    "/exp/lesson/{id}": {
      "get": {
        "operationId": "GetLesson",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expLesson"
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
          "LessonService"
        ]
      }
    },
    "/exp/lesson/{id}/prereqs": {
      "get": {
        "operationId": "GetAllLessonPrereqs",
        "responses": {
          "200": {
            "description": "A successful response.",
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
          "LessonService"
        ]
      }
    }
  },
  "definitions": {
    "expConnection": {
      "type": "object",
      "properties": {
        "A": {
          "type": "string"
        },
        "B": {
          "type": "string"
        }
      }
    },
    "expEndpoint": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Image": {
          "type": "string"
        },
        "configurationType": {
          "type": "string"
        },
        "Ports": {
          "type": "array",
          "items": {
            "type": "integer",
            "format": "int32"
          }
        },
        "Presentations": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expPresentation"
          }
        }
      }
    },
    "expLesson": {
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
        "Endpoints": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expEndpoint"
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
          "type": "integer",
          "format": "int32"
        },
        "Description": {
          "type": "string"
        },
        "Slug": {
          "type": "string",
          "title": "This is meant to fill: \"How well do you know \u003cslug\u003e?\""
        },
        "LessonFile": {
          "type": "string"
        },
        "LessonDir": {
          "type": "string"
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
        },
        "VerifyCompleteness": {
          "type": "boolean",
          "format": "boolean"
        },
        "VerifyObjective": {
          "type": "string"
        }
      }
    },
    "expLessons": {
      "type": "object",
      "properties": {
        "lessons": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLesson"
          }
        }
      }
    },
    "expPresentation": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Port": {
          "type": "integer",
          "format": "int32"
        },
        "Type": {
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
            "description": "A successful response.",
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
            "description": "A successful response.",
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
            "description": "A successful response.",
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
    "/exp/livelesson/{id}/verify": {
      "post": {
        "operationId": "RequestVerification",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expVerificationTaskUUID"
            }
          }
        },
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/expLessonUUID"
            }
          }
        ],
        "tags": [
          "LiveLessonsService"
        ]
      }
    },
    "/exp/verification/{id}": {
      "get": {
        "operationId": "GetVerification",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expVerificationTask"
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
    "expKillLiveLessonStatus": {
      "type": "object",
      "properties": {
        "success": {
          "type": "boolean",
          "format": "boolean"
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
    "expLiveLessons": {
      "type": "object",
      "properties": {
        "items": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveLesson"
          }
        }
      }
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
    },
    "expVerificationTask": {
      "type": "object",
      "properties": {
        "liveLessonId": {
          "type": "string"
        },
        "liveLessonStage": {
          "type": "integer",
          "format": "int32"
        },
        "success": {
          "type": "boolean",
          "format": "boolean"
        },
        "working": {
          "type": "boolean",
          "format": "boolean"
        },
        "message": {
          "type": "string"
        },
        "completed": {
          "type": "string",
          "format": "date-time"
        }
      }
    },
    "expVerificationTaskUUID": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string"
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
            "description": "A successful response.",
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
