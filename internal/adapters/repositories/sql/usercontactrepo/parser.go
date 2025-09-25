package usercontactrepo

import "go.openfort.xyz/shield/internal/core/domain/usercontact"

type parser struct {
}

func newParser() *parser {
	return &parser{}
}

func (p *parser) toDomain(userContact *UserContact) *usercontact.UserContact {
	return &usercontact.UserContact{
		ID:             userContact.ID,
		ExternalUserID: userContact.ExternalUserID,
		Email:          userContact.Email,
		Phone:          userContact.Phone,
	}
}

func (p *parser) toDatabase(userContact *usercontact.UserContact) *UserContact {
	return &UserContact{
		ID:             userContact.ID,
		ExternalUserID: userContact.ExternalUserID,
		Email:          userContact.Email,
		Phone:          userContact.Phone,
	}
}
