package middlewares

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uc-cdis/cohort-middleware/models"
	"github.com/uc-cdis/cohort-middleware/utils"
)

type TeamProjectAuthzI interface {
	TeamProjectValidationForCohort(ctx *gin.Context, cohortDefinitionId int) bool
	TeamProjectValidation(ctx *gin.Context, cohortDefinitionId int, filterCohortPairs []utils.CustomDichotomousVariableDef) bool
}

type TeamProjectAuthz struct {
	cohortDefinitionModel models.CohortDefinitionI
}

func NewTeamProjectAuthz(cohortDefinitionModel models.CohortDefinitionI) TeamProjectAuthz {
	return TeamProjectAuthz{cohortDefinitionModel: cohortDefinitionModel}
}
func hasAccessToAtLeastOne(ctx *gin.Context, teamProjects []string) bool {

	// query Arborist and return as soon as one of the teamProjects access check returns 200:
	for _, teamProject := range teamProjects {
		teamProjectAsResourcePath := teamProject
		teamProjectAccessService := "atlas-argo-wrapper-and-cohort-middleware"

		req, err := PrepareNewArboristRequestForResourceAndService(ctx, teamProjectAsResourcePath, teamProjectAccessService)
		if err != nil {
			ctx.AbortWithStatus(500)
			panic("Error while preparing Arborist request")
		}
		client := &http.Client{}
		// send the request to Arborist:
		resp, _ := client.Do(req)

		// arborist will return with 200 if the user has been granted access to the cohort-middleware URL in ctx:
		if resp.StatusCode == 200 {
			return true
		} else {
			// unauthorized or otherwise:
			log.Printf("Got response status %d from Arborist...", resp.StatusCode)
		}
	}
	return false
}

func (u TeamProjectAuthz) TeamProjectValidationForCohort(ctx *gin.Context, cohortDefinitionId int) bool {
	filterCohortPairs := []utils.CustomDichotomousVariableDef{}
	return u.TeamProjectValidation(ctx, cohortDefinitionId, filterCohortPairs)
}

// "team project" related checks:
// (1) check if the request contains any cohorts and if all cohorts belong to the same "team project"
// (2) check if the user has permission in the "team project"
// Returns true if both checks above pass, false otherwise.
func (u TeamProjectAuthz) TeamProjectValidation(ctx *gin.Context, cohortDefinitionId int, filterCohortPairs []utils.CustomDichotomousVariableDef) bool {

	uniqueCohortDefinitionIdsList := utils.GetUniqueCohortDefinitionIdsListFromRequest(cohortDefinitionId, filterCohortPairs)
	teamProjects, _ := u.cohortDefinitionModel.GetTeamProjectsThatMatchAllCohortDefinitionIds(uniqueCohortDefinitionIdsList)
	if len(teamProjects) == 0 {
		log.Printf("Invalid request error: could not find a 'team project' that is associated to ALL the cohorts present in this request")
		return false
	}
	if !hasAccessToAtLeastOne(ctx, teamProjects) {
		log.Printf("Invalid request error: user does not have access to any of the 'team projects' associated with the cohorts in this request")
		return false
	}
	// passed both tests:
	return true
}
