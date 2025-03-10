package transform

import (
	"errors"

	"github.com/navimar-systems/goasterix"
)

var (
	ErrCartOrdUnknown = errors.New("[ASTERIX Error] CART ORD Unknown")
)

type BiaisRadar struct {
	SacSic        SourceIdentifier `json:"sourceIdentifier"`
	GainDistance  float64          `json:"gainDistance"`
	BiaisDistance float64          `json:"biaisDistance"`
	BiaisAzimut   float64          `json:"biaisAzimut"`
	BiaisDatation float64          `json:"biaisDatation"`
}
type CarteActive struct {
	Nom string `json:"nom"`
	Ord string `json:"ord"`
}
type NivC struct {
	NivInf int16 `json:"nivinf"`
	NivSup int16 `json:"nivsup"`
}

type PresenceSTPV struct {
	Version uint8  `json:"version"`
	Nap     uint8  `json:"nap"`
	NS      string `json:"ns"`
	ST      string `json:"st,omitempty"`
	PS      string `json:"ps,omitempty"`
}

type Cat255STRModel struct {
	SacSic *SourceIdentifier `json:"SourceIdentifier,omitempty"`
	Hem    float64           `json:"hem,omitempty"`
	Spe    *PresenceSTPV     `json:"spe,omitempty"`
	Nivc   *NivC             `json:"nivc,omitempty"`
	Txtc   string            `json:"txtc,omitempty"`
	Cart   *CarteActive      `json:"cart,omitempty"`
	Biais  []BiaisRadar      `json:"biais,omitempty"`
}

func (data *Cat255STRModel) write(rec goasterix.Record) {
	for _, item := range rec.Items {
		switch item.Meta.FRN {
		case 1:
			// decode sac sic
			var payload [2]byte
			copy(payload[:], item.Fixed.Data[:])
			tmp, _ := sacSic(payload)
			data.SacSic = &tmp
		case 2:
			// HEM : Heure d’émission du message d’alerte
			var payload [3]byte
			copy(payload[:], item.Fixed.Data[:])
			data.Hem, _ = timeOfDay(payload)
		case 3:
			// SPE : Présence STR-STPV
			tmp := speStpv(*item.Extended)
			data.Spe = &tmp
		case 4:
			// NIVC : Niveaux optionnels assignés à la carte dynamique
			var payload [4]byte
			copy(payload[:], item.Fixed.Data[:])
			tmp := nivCarte(payload)
			data.Nivc = &tmp
		case 5:
			// TXTC : Texte optionnel de la carte dynamique
			data.Txtc = string(item.Repetitive.Data)
		case 6:
			// CART : activation de cartes dynamiques
			var payload [9]byte
			copy(payload[:], item.Fixed.Data[:])
			tmp, _ := carte(payload)
			data.Cart = &tmp
		case 7:
			// BIAIS : Valeurs des biais courants radars
			data.Biais = biaisExtract(*item.Repetitive)
		}
	}
}

func speStpv(item goasterix.Extended) PresenceSTPV {
	var spe PresenceSTPV
	spe.Version = item.Primary[0] & 0xE0 >> 5
	spe.Nap = item.Primary[0] & 0x18 >> 3

	tmpNs := item.Primary[0] & 0x06 >> 1
	switch tmpNs {
	case 0:
		spe.NS = "principal"
	case 1:
		spe.NS = "secours"
	case 2:
		spe.NS = "test"
	}

	if item.Secondary != nil {
		if item.Secondary[0]&0x80 != 0 {
			spe.ST = "evaluation"
		} else {
			spe.ST = "operational"
		}
		if item.Secondary[0]&0x40 != 0 {
			spe.PS = "stpv_deconnecte_str"
		} else {
			spe.PS = "stpv_connecte_str"
		}
	}

	return spe
}

func nivCarte(data [4]byte) NivC {
	var nivc NivC
	nivc.NivInf = int16(data[0])<<8 + int16(data[1])
	nivc.NivSup = int16(data[2])<<8 + int16(data[3])
	return nivc
}

func carte(data [9]byte) (CarteActive, error) {
	var cart CarteActive
	var err error
	cart.Nom = string(data[:8])
	tmpOrd := data[8] & 0xE0 >> 5
	switch tmpOrd {
	case 0:
		cart.Ord = "activation_carte"
	case 1:
		cart.Ord = "annulation_carte"
	default:
		cart.Ord = "unknowm"
		err = ErrCartOrdUnknown
	}
	return cart, err
}

func biaisExtract(item goasterix.Repetitive) []BiaisRadar {
	var biais []BiaisRadar
	n := int(item.Rep)
	data := item.Data
	for i := 0; i < n; i++ {
		b := BiaisRadar{}
		var sacsic [2]byte
		copy(sacsic[:], data[i:i+2])
		b.SacSic, _ = sacSic(sacsic)
		b.GainDistance = float64(uint16(data[i+2])<<8+uint16(data[i+3])) / 6384
		b.BiaisDistance = float64(int16(data[i+4])<<8 + int16(data[i+5]))
		b.BiaisAzimut = float64(int16(data[i+6])<<8+int16(data[i+7])) * 0.0055
		b.BiaisDatation = float64(int16(data[i+8])<<8+int16(data[i+9])) / 1024
		biais = append(biais, b)
	}
	return biais
}
