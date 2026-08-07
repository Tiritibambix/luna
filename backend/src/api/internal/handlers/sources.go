package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"luna-backend/api/internal/util"
	"luna-backend/constants"
	"luna-backend/errors"
	"luna-backend/files"
	"luna-backend/protocols/caldav"
	"luna-backend/protocols/google"
	"luna-backend/protocols/ical"
	"luna-backend/types"

	"github.com/gin-gonic/gin"
)

type exposedSource struct {
	Id              types.ID `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	CanAddCalendars bool     `json:"can_add_calendars"`
}

type exposedDetailedSource struct {
	Id              types.ID `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Settings        any      `json:"settings,omitempty"`
	AuthType        string   `json:"auth_type"`
	Auth            any      `json:"auth"`
	CanAddCalendars bool     `json:"can_add_calendars"`
}

func getSources(u *util.HandlerUtility, userId types.ID) ([]types.Source, *errors.ErrorTrace) {
	srcs, err := u.Tx.Queries().GetSourcesByUser(userId, u.Context, u.Config)
	if err != nil {
		return nil, err
	}
	return srcs, nil
}

func GetSources(c *gin.Context) {
	u := util.GetUtil(c)

	userId := util.GetUserId(c)

	sources, err := getSources(u, userId)
	if err != nil {
		u.Error(err)
		return
	}

	exposedSources := make([]exposedSource, len(sources))
	for i, source := range sources {
		u.Config.Cache.Cache(userId, source)

		exposedSources[i] = exposedSource{
			Id:              source.GetId(),
			Name:            source.GetName(),
			Type:            source.GetType(),
			CanAddCalendars: source.CanAddCalendars(),
		}
	}

	u.Success(&gin.H{"sources": exposedSources})
}

func GetSource(c *gin.Context) {
	u := util.GetUtil(c)

	sourceId, err := util.GetId(c, "source")
	if err != nil {
		u.Error(err)
		return
	}

	userId := util.GetUserId(c)

	source, err := u.Tx.Queries().GetSource(userId, sourceId, u.Context, u.Config)
	if err != nil {
		u.Error(err)
		return
	}

	u.Config.Cache.Cache(userId, source)

	exposedSource := exposedDetailedSource{
		Id:              source.GetId(),
		Name:            source.GetName(),
		Settings:        source.GetSettings(),
		Type:            source.GetType(),
		AuthType:        source.GetAuth().GetType(),
		Auth:            source.GetAuth(),
		CanAddCalendars: source.CanAddCalendars(),
	}

	u.Success(&gin.H{"source": exposedSource})
}

func parseSource(c *gin.Context, sourceType string, sourceName string, sourceAuth types.AuthMethod, sourceSettings types.SourceSettings, user types.ID, q types.DatabaseQueries, ctx context.Context) (types.Source, *errors.ErrorTrace) {
	var tr *errors.ErrorTrace
	var source types.Source

	switch sourceType {
	case constants.SourceCaldav:
		caldavSettings := sourceSettings.(*caldav.CaldavSourceSettings)
		if caldavSettings.Url == nil {
			return nil, errors.New().Status(http.StatusBadRequest).
				Append(errors.LvlPlain, "Missing CalDAV url")
		}
		source = caldav.NewCaldavSource(sourceName, caldavSettings.Url, sourceAuth)
		source.SupplyContext(ctx)

	case constants.SourceIcal:
		icalSettings := sourceSettings.(*ical.IcalSourceSettings)
		switch icalSettings.Location {
		case "remote":
			if icalSettings.Url == nil {
				return nil, errors.New().Status(http.StatusBadRequest).
					Append(errors.LvlPlain, "Missing iCal URL")
			}
			source, tr = ical.NewRemoteIcalSource(sourceName, icalSettings.Url, sourceAuth, user, q)
			if tr != nil {
				return nil, tr
			}
		case "local":
			if sourceAuth.GetType() != constants.AuthNone {
				return nil, errors.New().Status(http.StatusBadRequest).
					Append(errors.LvlPlain, "Local iCal sources do not support authentication")
			}
			if icalSettings.Path == nil {
				return nil, errors.New().Status(http.StatusBadRequest).
					Append(errors.LvlPlain, "Missing iCal Path")
			}
			source = ical.NewLocalIcalSource(sourceName, icalSettings.Path)
		case "database":
			if sourceAuth.GetType() != constants.AuthNone {
				return nil, errors.New().Status(http.StatusBadRequest).
					Append(errors.LvlPlain, "Database iCal sources do not support authentication")
			}

			if icalSettings.File == nil {
				return nil, errors.New().Status(http.StatusBadRequest).
					Append(errors.LvlPlain, "Missing iCal file")
			}

			file, err := icalSettings.File.Open()
			if err != nil {
				return nil, errors.New().Status(http.StatusInternalServerError).
					AddErr(errors.LvlDebug, err).
					Append(errors.LvlPlain, "Could not open iCal file")
			}

			var contentToSave bytes.Buffer
			contentToValidate := io.TeeReader(file, &contentToSave)

			fileParseErr := files.IsValidIcalFile(contentToValidate)
			if fileParseErr != nil {
				return nil, fileParseErr
			}

			var tr *errors.ErrorTrace
			source, tr = ical.NewDatabaseIcalSource(sourceName, icalSettings.File.Filename, &contentToSave, user, q)
			if tr != nil {
				return nil, tr
			}
		case "":
			return nil, errors.New().Status(http.StatusBadRequest).
				Append(errors.LvlPlain, "Missing iCal location")
		default:
			return nil, errors.New().Status(http.StatusBadRequest).
				Append(errors.LvlPlain, "Unknown iCal location: %v", icalSettings.Location)
		}

	case constants.SourceGoogle:
		source = google.NewGoogleSource(sourceName, sourceAuth)

	case "":
		return nil, errors.New().Status(http.StatusBadRequest).
			Append(errors.LvlPlain, "Missing source type")
	default:
		return nil, errors.New().Status(http.StatusBadRequest).
			Append(errors.LvlPlain, "Unknown source type: %v", sourceType)
	}

	return source, nil
}

func PutSource(c *gin.Context, body *struct {
	Name     string `json:"name" form:"name" binding:"required"`
	Type     string `json:"type" form:"type" binding:"required"`
	AuthType string `json:"auth_type" form:"auth_type" binding:"required"`
}) {
	u := util.GetUtil(c)

	userId := util.GetUserId(c)

	if body.Name == "" {
		u.Error(errors.New().Status(http.StatusBadRequest).
			Append(errors.LvlPlain, "Missing name"))
		return
	}

	sourceAuth, tr := util.ParseAuth(c, body.AuthType, userId)
	if tr != nil {
		u.Error(tr)
		return
	}

	sourceSettings, tr := util.ParseSourceSettings(c, body.Type)
	if tr != nil {
		u.Error(tr)
		return
	}

	source, tr := parseSource(c, body.Type, body.Name, sourceAuth, sourceSettings, userId, u.Tx.Queries(), u.Context)
	if tr != nil {
		u.Error(tr)
		return
	}

	id, tr := u.Tx.Queries().InsertSource(userId, source)
	if tr != nil {
		u.Error(tr)
		return
	}

	u.Success(&gin.H{"id": id.String()})
}

func PatchSource(c *gin.Context, body *struct {
	Name     *string `json:"name" form:"name"`
	Type     *string `json:"type" form:"type"`
	AuthType *string `json:"auth_type" form:"auth_type"`
}) {
	u := util.GetUtil(c)

	userId := util.GetUserId(c)

	sourceId, tr := util.GetId(c, "source")
	if tr != nil {
		u.Error(tr)
		return
	}

	source, tr := u.Tx.Queries().GetSource(userId, sourceId, u.Context, u.Config)
	if tr != nil {
		u.Error(tr)
		return
	}

	var newAuth types.AuthMethod = nil
	if body.AuthType != nil {
		newAuth, tr = util.ParseAuth(c, *body.AuthType, userId)
		if tr != nil {
			u.Error(tr)
			return
		}
	}

	var newSourceSettings types.SourceSettings = nil
	if body.Type != nil {
		newSourceSettings, tr = util.ParseSourceSettings(c, *body.Type)
		if tr != nil {
			u.Error(tr)
			return
		}

		tr = source.Cleanup(u.Tx.Queries())
		if tr != nil {
			u.Error(tr.
				Append(errors.LvlWordy, "Could not clean up source before editing"))
			return
		}
	}

	tr = u.Tx.Queries().UpdateSource(userId, sourceId, body.Name, newAuth, body.Type, newSourceSettings)
	if tr != nil {
		u.Error(tr)
		return
	}

	u.Success(nil)
}

func DeleteSource(c *gin.Context) {
	u := util.GetUtil(c)

	userId := util.GetUserId(c)

	sourceId, err := util.GetId(c, "source")
	if err != nil {
		u.Error(err)
		return
	}

	source, err := u.Tx.Queries().GetSource(userId, sourceId, u.Context, u.Config)
	if err != nil {
		u.Warn(err)
	} else {
		err = source.Cleanup(u.Tx.Queries())
		if err != nil {
			u.Warn(err.
				Append(errors.LvlWordy, "Could not clean up source before deleting"))
		}
	}

	deleted, err := u.Tx.Queries().DeleteSource(userId, sourceId)
	if err != nil {
		u.Error(err)
		return
	}

	if deleted {
		u.Success(nil)
	} else {
		u.Error(errors.New().Status(http.StatusNotFound).
			Append(errors.LvlPlain, "Source not found"))
	}
}

func ChangeSourceDisplayOrder(c *gin.Context, body *struct {
	Index *uint16 `json:"index" form:"index" binding:"required"`
}) {
	u := util.GetUtil(c)

	userId := util.GetUserId(c)

	sourceId, tr := util.GetId(c, "source")
	if tr != nil {
		u.Error(tr)
		return
	}

	tr = u.Tx.Queries().UpdateSourceDisplayOrder(userId, sourceId, *body.Index)
	if tr != nil {
		u.Error(tr)
		return
	}

	u.Success(nil)
}
