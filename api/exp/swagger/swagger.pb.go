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
    "/exp/collection/{slug}": {
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
        "lessonSlug": {
          "type": "string"
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
Image = `{
  "swagger": "2.0",
  "info": {
    "title": "image.proto",
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
    "/exp/lesson/{slug}": {
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
              "$ref": "#/definitions/expLBHealthCheckResponse"
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
    "expLiveLesson": {
      "type": "object",
      "properties": {
        "Id": {
          "type": "string"
        },
        "LessonSlug": {
          "type": "string"
        },
        "LiveEndpoints": {
          "type": "object",
          "additionalProperties": {
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
        },
        "HealthyTests": {
          "type": "integer",
          "format": "int32"
        },
        "TotalTests": {
          "type": "integer",
          "format": "int32"
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
        "items": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/expLiveLesson"
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
Livesession = `{
  "swagger": "2.0",
  "info": {
    "title": "livesession.proto",
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
        "Id": {
          "type": "string"
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
        },
        "imageVersion": {
          "type": "string"
        }
      }
    }
  }
}
`
)
