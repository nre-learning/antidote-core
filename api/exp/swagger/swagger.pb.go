package swagger 

const (
Antidoteinfo = `{
  "swagger": "2.0",
  "info": {
    "title": "antidoteinfo.proto",
    "version": "version not set"
  },
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/exp/antidoteinfo": {
      "get": {
        "operationId": "GetAntidoteInfo",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expAntidoteInfo"
            }
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
            }
          }
        },
        "tags": [
          "AntidoteInfoService"
        ]
      }
    }
  },
  "definitions": {
    "expAntidoteInfo": {
      "type": "object",
      "properties": {
        "buildSha": {
          "type": "string"
        },
        "buildVersion": {
          "type": "string"
        },
        "curriculumVersion": {
          "type": "string"
        }
      }
    },
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "runtimeError": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string"
        },
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
          }
        }
      }
    }
  }
}
`
Collection = `{
  "swagger": "2.0",
  "info": {
    "title": "collection.proto",
    "version": "version not set"
  },
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
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
            }
          }
        },
        "tags": [
          "CollectionService"
        ]
      }
    },
    "/exp/collection/{slug}": {
      "get": {
        "operationId": "GetCollection",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expCollection"
            }
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
            }
          }
        },
        "parameters": [
          {
            "name": "slug",
            "in": "path",
            "required": true,
            "type": "string"
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
        "Slug": {
          "type": "string"
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
        "Slug": {
          "type": "string"
        },
        "Name": {
          "type": "string"
        },
        "Description": {
          "type": "string"
        }
      }
    },
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "runtimeError": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string"
        },
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
          }
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
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
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
    },
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "runtimeError": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string"
        },
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
          }
        }
      }
    }
  }
}
`
Image = `{
  "swagger": "2.0",
  "info": {
    "title": "image.proto",
    "version": "version not set"
  },
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {},
  "definitions": {
    "expImage": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        }
      }
    },
    "expImages": {
      "type": "object",
      "properties": {
        "items": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expImage"
          }
        }
      }
    },
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "runtimeError": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string"
        },
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
          }
        }
      }
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
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
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
    "/exp/lesson/{slug}": {
      "get": {
        "operationId": "GetLesson",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expLesson"
            }
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
            }
          }
        },
        "parameters": [
          {
            "name": "slug",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "tags": [
          "LessonService"
        ]
      }
    },
    "/exp/lesson/{slug}/prereqs": {
      "get": {
        "summary": "NOTE that this doesn't just get the prereqs for this lesson, but for all dependent\nlessons as well. So it's not enough to just retrieve from the prereqs field in a given lesson,\nthis function will traverse that tree for you and provide a flattened and de-duplicated list.",
        "operationId": "GetAllLessonPrereqs",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expLessonPrereqs"
            }
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
            }
          }
        },
        "parameters": [
          {
            "name": "slug",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "tags": [
          "LessonService"
        ]
      }
    }
  },
  "definitions": {
    "expAuthor": {
      "type": "object",
      "properties": {
        "Name": {
          "type": "string"
        },
        "Link": {
          "type": "string"
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
        "ConfigurationType": {
          "type": "string"
        },
        "AdditionalPorts": {
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
        },
        "Host": {
          "type": "string"
        }
      }
    },
    "expLesson": {
      "type": "object",
      "properties": {
        "Slug": {
          "type": "string"
        },
        "Stages": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLessonStage"
          }
        },
        "Name": {
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
        "Authors": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expAuthor"
          }
        },
        "Category": {
          "type": "string"
        },
        "Diagram": {
          "type": "string"
        },
        "Video": {
          "type": "string"
        },
        "Tier": {
          "type": "string"
        },
        "Prereqs": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "title": "this field ONLY contains immediately listed prereqs from the lesson meta.\nfor a full flattened tree of all prereqs, see GetAllLessonPrereqs"
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
        "ShortDescription": {
          "type": "string",
          "title": "This is meant to fill: \"How well do you know \u003cShortDescription\u003e?\""
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
            "type": "string"
          }
        }
      }
    },
    "expLessonStage": {
      "type": "object",
      "properties": {
        "Description": {
          "type": "string"
        },
        "GuideType": {
          "type": "string"
        },
        "StageVideo": {
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
    },
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "runtimeError": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string"
        },
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
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
              "$ref": "#/definitions/expLBHealthCheckResponse"
            }
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
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
              "$ref": "#/definitions/expLiveLessonId"
            }
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
            }
          }
        },
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/expLiveLessonRequest"
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
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
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
    "expKillLiveLessonStatus": {
      "type": "object",
      "properties": {
        "success": {
          "type": "boolean",
          "format": "boolean"
        }
      }
    },
    "expLBHealthCheckResponse": {
      "type": "object"
    },
    "expLiveEndpoint": {
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
        },
        "LivePresentations": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/expLivePresentation"
          }
        },
        "Host": {
          "type": "string"
        },
        "SSHUser": {
          "type": "string"
        },
        "SSHPassword": {
          "type": "string"
        }
      }
    },
    "expLiveLesson": {
      "type": "object",
      "properties": {
        "ID": {
          "type": "string"
        },
        "SessionID": {
          "type": "string"
        },
        "AntidoteID": {
          "type": "string"
        },
        "LessonSlug": {
          "type": "string"
        },
        "LiveEndpoints": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveEndpoint"
          }
        },
        "CurrentStage": {
          "type": "integer",
          "format": "int32"
        },
        "GuideContents": {
          "type": "string"
        },
        "GuideType": {
          "type": "string"
        },
        "Status": {
          "type": "string"
        },
        "Error": {
          "type": "boolean",
          "format": "boolean"
        },
        "HealthyTests": {
          "type": "integer",
          "format": "int32"
        },
        "TotalTests": {
          "type": "integer",
          "format": "int32"
        },
        "Diagram": {
          "type": "string"
        },
        "Video": {
          "type": "string"
        },
        "StageVideo": {
          "type": "string"
        }
      }
    },
    "expLiveLessonId": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string"
        }
      }
    },
    "expLiveLessonRequest": {
      "type": "object",
      "properties": {
        "lessonSlug": {
          "type": "string"
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
    "expLiveLessons": {
      "type": "object",
      "properties": {
        "LiveLessons": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveLesson"
          }
        }
      }
    },
    "expLivePresentation": {
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
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "runtimeError": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string"
        },
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
          }
        }
      }
    }
  }
}
`
Livesession = `{
  "swagger": "2.0",
  "info": {
    "title": "livesession.proto",
    "version": "version not set"
  },
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/exp/livesession": {
      "post": {
        "summary": "Request a lab is created, or request the UUID of one that already exists for these parameters.",
        "operationId": "RequestLiveSession",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/expLiveSession"
            }
          },
          "default": {
            "description": "An unexpected error response",
            "schema": {
              "$ref": "#/definitions/runtimeError"
            }
          }
        },
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "properties": {}
            }
          }
        ],
        "tags": [
          "LiveSessionsService"
        ]
      }
    }
  },
  "definitions": {
    "expLiveSession": {
      "type": "object",
      "properties": {
        "ID": {
          "type": "string"
        },
        "SourceIP": {
          "type": "string"
        },
        "Persistent": {
          "type": "boolean",
          "format": "boolean"
        }
      }
    },
    "expLiveSessions": {
      "type": "object",
      "properties": {
        "items": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveSession"
          }
        }
      }
    },
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "runtimeError": {
      "type": "object",
      "properties": {
        "error": {
          "type": "string"
        },
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
          }
        }
      }
    }
  }
}
`
)
