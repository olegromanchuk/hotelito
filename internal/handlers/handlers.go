package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/pbx"
	"github.com/sirupsen/logrus"
	"net/http"
)

type Handler struct {
	Log   *logrus.Logger
	PBX   pbx.PBXProvider
	Hotel hotel.HospitalityProvider
}

func NewHandler(log *logrus.Logger, pbx pbx.PBXProvider, hotel hotel.HospitalityProvider) *Handler {
	return &Handler{
		Log:   log,
		PBX:   pbx,
		Hotel: hotel,
	}
}

func (h *Handler) HandleManualLogin(w http.ResponseWriter, r *http.Request) {
	url, err := h.Hotel.HandleInitialLogin()
	if err != nil {
		h.Log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			h.Log.Error(err)
			return
		}
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	h.Log.Debugf("Handling callback")
	state := r.FormValue("state")
	code := r.FormValue("code")
	err := h.Hotel.HandleOAuthCallback(state, code)
	if err != nil {
		h.Log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			return
		}
		return
	}
	h.Log.Debugf("Got auth code: %s state: %s", code, state)
	h.Log.Infof("Ready for future requests")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Great Success! Ready for future requests. You can close this window now."))
	if err != nil {
		h.Log.Error(err)
		return
	}
}

func (h *Handler) Handle3cxCallInfo(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	//h.Log.Debugf("Received 3cx call info")
	room, err := h.PBX.ProcessPBXRequest(decoder)
	if err != nil {
		if err.Error() == "incoming-call-ignoring" { //ignore incoming calls
			return
		}
		h.Log.Error(err)
		return
	}
	if room.PhoneNumber == "" {
		h.Log.Error("Room phone number is empty")
		return
	}
	h.Log.Debugf("Room phone number: %s", room.PhoneNumber)

	//get provider
	hotelProvider := h.Hotel
	msg, err := hotelProvider.UpdateRoom(room.PhoneNumber, room.RoomCondition, room.HousekeeperName)
	if err != nil {
		h.Log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			h.Log.Error(err)
			return
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(msg))
	if err != nil {
		h.Log.Error(err)
		return
	}
}

func (h *Handler) Handle3cxLookup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	number := query.Get("Number")

	jsonAsBytes, err := h.PBX.ProcessLookupByNumber(number) //returns dummy contact with "number"
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonAsBytes)
	if err != nil {
		h.Log.Error(err)
		return
	}

}

func (h *Handler) HandleSetHousekeepingStatus(w http.ResponseWriter, r *http.Request) {
	h.Log.Debugf("HandleSetHousekeepingStatus")

	// Get the housekeeping info from the URL
	vars := mux.Vars(r)
	roomPhoneNumber := vars["roomPhoneNumber"]
	housekeepingStatus := vars["housekeepingStatus"]
	housekeeperID := vars["housekeeperID"]

	//get provider

	h.Log.Debugf("roomPhoneNumber: %s, housekeepingStatus: %s, housekeeperID: %s", roomPhoneNumber, housekeepingStatus, housekeeperID)
	hotelProvider := h.Hotel
	//roomPhoneNumber = 1001
	msg, err := hotelProvider.UpdateRoom(roomPhoneNumber, housekeepingStatus, housekeeperID)
	if err != nil {
		h.Log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			h.Log.Error(err)
			return
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(msg))
	if err != nil {
		h.Log.Error(err)
		return
	}
}

func (h *Handler) HandleGetRooms(w http.ResponseWriter, r *http.Request) {
	h.Log.Debugf("HandleGetRooms")

	hotelProvider := h.Hotel
	rooms, err := hotelProvider.GetRooms()
	if err != nil {
		h.Log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			h.Log.Error(err)
			return
		}
		return
	}
	msg := fmt.Sprintf("amount of rooms: %d", len(rooms))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(msg))
	if err != nil {
		h.Log.Error(err)
		return
	}
}

func (h *Handler) HandleMain(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<a href="/login">Login with OAuth2 Provider</a>`)
}
