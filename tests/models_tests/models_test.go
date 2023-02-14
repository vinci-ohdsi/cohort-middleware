package models_tests

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/uc-cdis/cohort-middleware/config"
	"github.com/uc-cdis/cohort-middleware/db"
	"github.com/uc-cdis/cohort-middleware/models"
	"github.com/uc-cdis/cohort-middleware/tests"
	"github.com/uc-cdis/cohort-middleware/utils"
	"github.com/uc-cdis/cohort-middleware/version"
)

var testSourceId = tests.GetTestSourceId()
var allCohortDefinitions []*models.CohortDefinitionStats
var smallestCohort *models.CohortDefinitionStats
var largestCohort *models.CohortDefinitionStats
var secondLargestCohort *models.CohortDefinitionStats
var extendedCopyOfSecondLargestCohort *models.CohortDefinitionStats
var thirdLargestCohort *models.CohortDefinitionStats
var allConceptIds []int64
var genderConceptId = tests.GetTestGenderConceptId()
var hareConceptId = tests.GetTestHareConceptId()
var asnHareConceptId = tests.GetTestAsnHareConceptId()
var histogramConceptId = tests.GetTestHistogramConceptId()

func TestMain(m *testing.M) {
	setupSuite()
	retCode := m.Run()
	tearDownSuite()
	os.Exit(retCode)
}

func setupSuite() {
	log.Println("setup for suite")
	// connect to test db:
	config.Init("development")
	db.Init()
	// ensure we start w/ empty db:
	tearDownSuite()
	// load test seed data, including test cohorts referenced below:
	tests.ExecSQLScript("../setup_local_db/test_data_results_and_cdm.sql", testSourceId)

	// initialize some handy variables to use in tests below:
	// (see also tests/setup_local_db/test_data_results_and_cdm.sql for these test cohort details)
	allCohortDefinitions, _ = cohortDefinitionModel.GetAllCohortDefinitionsAndStatsOrderBySizeDesc(testSourceId)
	largestCohort = allCohortDefinitions[0]
	secondLargestCohort = allCohortDefinitions[2]
	extendedCopyOfSecondLargestCohort = allCohortDefinitions[1]
	thirdLargestCohort = allCohortDefinitions[3]
	smallestCohort = allCohortDefinitions[len(allCohortDefinitions)-1]
	concepts, _ := conceptModel.RetriveAllBySourceId(testSourceId)
	allConceptIds = tests.MapIntAttr(concepts, "ConceptId")
}

func tearDownSuite() {
	log.Println("teardown for suite")
	tests.ExecAtlasSQLScript("../setup_local_db/ddl_atlas.sql")
	// we need some basic atlas data in "source" table to be able to connect to results DB, and this script has it:
	tests.ExecAtlasSQLScript("../setup_local_db/test_data_atlas.sql")
	tests.ExecSQLScript("../setup_local_db/ddl_results_and_cdm.sql", testSourceId)
}

func setUp(t *testing.T) {
	log.Println("setup for test")

	// ensure tearDown is called when test "t" is done:
	t.Cleanup(func() {
		tearDown()
	})
}

func tearDown() {
	log.Println("teardown for test")
}

var conceptModel = new(models.Concept)
var cohortDefinitionModel = new(models.CohortDefinition)
var cohortDataModel = new(models.CohortData)

var versionModel = new(models.Version)
var sourceModel = new(models.Source)

func TestGetConceptId(t *testing.T) {
	setUp(t)
	conceptId := models.GetConceptId("ID_12345")
	if conceptId != 12345 {
		t.Error()
	}
	// the GetConceptId below should result in panic/error:
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	models.GetConceptId("AD_12345")

}

func TestGetPrefixedConceptId(t *testing.T) {
	setUp(t)
	conceptId := models.GetPrefixedConceptId(12345)
	if conceptId != "ID_12345" {
		t.Error()
	}
}

func TestGetConceptValueNotNullCheckBasedOnConceptTypeError(t *testing.T) {
	setUp(t)
	// the call below should result in panic/error:
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	models.GetConceptValueNotNullCheckBasedOnConceptType("observation", testSourceId, -1)
}

func TestGetConceptValueNotNullCheckBasedOnConceptTypeError2(t *testing.T) {
	setUp(t)
	// add dummy concept:
	conceptId := tests.AddInvalidTypeConcept(models.Omop)

	// the call below should result in a specific panic/error on the concept type not being supported:
	defer func() {
		r := recover()
		if r == nil || !strings.HasPrefix(r.(string), "error: concept type not supported") {
			t.Errorf("The code did not panic with expected error")
		}
		// cleanup:
		tests.RemoveConcept(models.Omop, conceptId)
	}()
	models.GetConceptValueNotNullCheckBasedOnConceptType("observation", testSourceId, conceptId)
}

func TestGetConceptValueNotNullCheckBasedOnConceptTypeSuccess(t *testing.T) {
	setUp(t)
	// check success scenarios:
	result := models.GetConceptValueNotNullCheckBasedOnConceptType("observation", testSourceId, hareConceptId)
	if result != "observation.value_as_concept_id is not null and observation.value_as_concept_id != 0" {
		t.Errorf("Unexpected result. Found %s", result)
	}
	result = models.GetConceptValueNotNullCheckBasedOnConceptType("observation", testSourceId, histogramConceptId)
	if result != "observation.value_as_number is not null" {
		t.Errorf("Unexpected result. Found %s", result)
	}
}

func TestRetriveAllBySourceId(t *testing.T) {
	setUp(t)
	concepts, _ := conceptModel.RetriveAllBySourceId(testSourceId)
	if len(concepts) != 10 {
		t.Errorf("Found %d", len(concepts))
	}
}

func TestRetrieveStatsBySourceIdAndCohortIdAndConceptIds(t *testing.T) {
	setUp(t)
	conceptsStats, _ := conceptModel.RetrieveStatsBySourceIdAndCohortIdAndConceptIds(testSourceId,
		smallestCohort.Id,
		allConceptIds)
	// simple test: we expect stats for each valid conceptId, therefore the lists are
	//  expected to have the same lenght here:
	if len(conceptsStats) != len(allConceptIds) {
		t.Errorf("Found %d", len(conceptsStats))
	}
}

func TestRetrieveStatsBySourceIdAndCohortIdAndConceptIdsCheckRatio(t *testing.T) {
	setUp(t)
	filterIds := []int64{genderConceptId}
	conceptsStats, _ := conceptModel.RetrieveStatsBySourceIdAndCohortIdAndConceptIds(testSourceId,
		secondLargestCohort.Id,
		filterIds)
	// simple test: in the test data we keep the gender concept *missing* at a ratio of 1/3 for the largest cohort. Here
	// we check if the missing ratio calculation is working correctly:
	if len(conceptsStats) != 1 {
		t.Errorf("Found %d", len(conceptsStats))
	}
	if conceptsStats[0].NmissingRatio != 1.0/3.0 {
		t.Errorf("Found wrong ratio %f", conceptsStats[0].NmissingRatio)
	}
}

func TestRetrieveInfoBySourceIdAndConceptIds(t *testing.T) {
	setUp(t)
	conceptsInfo, _ := conceptModel.RetrieveInfoBySourceIdAndConceptIds(testSourceId,
		allConceptIds)
	// simple test: we expect info for each valid conceptId, therefore the lists are
	//  expected to have the same lenght here:
	if len(conceptsInfo) != len(allConceptIds) {
		t.Errorf("Found %d", len(conceptsInfo))
	}
}

func TestRetrieveInfoBySourceIdAndConceptTypes(t *testing.T) {
	setUp(t)
	// get all concepts:
	conceptsInfo, _ := conceptModel.RetrieveInfoBySourceIdAndConceptIds(testSourceId,
		allConceptIds)
	// simple test: we know that not all concepts have the same type in our test db, so
	// if we query on the type of a single concept, the result should
	// be a list where 1 =< size < len(allConceptIds):
	conceptTypes := []string{conceptsInfo[0].ConceptType}
	conceptsInfo, _ = conceptModel.RetrieveInfoBySourceIdAndConceptTypes(testSourceId,
		conceptTypes)
	if !(1 <= len(conceptsInfo) && len(conceptsInfo) < len(allConceptIds)) {
		t.Errorf("Found %d", len(conceptsInfo))
	}
}

func TestRetrieveInfoBySourceIdAndConceptIdNotFound(t *testing.T) {
	setUp(t)
	// get all concepts:
	conceptInfo, error := conceptModel.RetrieveInfoBySourceIdAndConceptId(testSourceId,
		-1)
	if conceptInfo != nil {
		t.Errorf("Did not expect to find data")
	}
	if error == nil {
		t.Errorf("Expected error")
	}
}

func TestRetrieveInfoBySourceIdAndConceptId(t *testing.T) {
	setUp(t)
	// get all concepts:
	conceptInfo, _ := conceptModel.RetrieveInfoBySourceIdAndConceptId(testSourceId,
		genderConceptId)
	if conceptInfo == nil {
		t.Errorf("Expected to find data")
	}
}

func TestRetrieveInfoBySourceIdAndConceptTypesWrongType(t *testing.T) {
	setUp(t)
	// simple test: invalid/non-existing type should return an empty list:
	conceptTypes := []string{"invalid type"}
	conceptsInfo, _ := conceptModel.RetrieveInfoBySourceIdAndConceptTypes(testSourceId,
		conceptTypes)
	if len(conceptsInfo) != 0 {
		t.Errorf("Found %d", len(conceptsInfo))
	}
}

func TestRetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairsNoResults(t *testing.T) {
	setUp(t)
	// empty:
	filterCohortPairs := []utils.CustomDichotomousVariableDef{}
	stats, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(testSourceId,
		smallestCohort.Id,
		allConceptIds, filterCohortPairs, allConceptIds[0])
	// none of the subjects has a value in all the concepts, so we expect len==0 here:
	if len(stats) != 0 {
		t.Errorf("Expected no results, found %d", len(stats))
	}
}

// Tests various scenarios for QueryFilterByCohortPairsHelper.
// These tests currently use pre-defined cohorts (see .sql loaded in  setupSuite()).
// A possible improvement / TODO could be to write more test utility functions
// to add specific test data on the fly. This could make some of the test code (like the tests here)
// more readable by having the test data and the test assertions close together. For now,
// consider reading these tests together with the .sql file that is loaded in setupSuite()
// to understand how the test cohorts relate to each other.
func TestQueryFilterByCohortPairsHelper(t *testing.T) {
	setUp(t)

	type SubjectId struct {
		SubjectId int
	}
	// smallestCohort and largestCohort do not overlap...
	filterCohortPairs := []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    smallestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
	}
	resultsDataSource := tests.GetResultsDataSource()
	var subjectIds []*SubjectId
	population := largestCohort
	query := models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// ...so we expect overlap the size of the largestCohort:
	if len(subjectIds) != largestCohort.CohortSize {
		t.Errorf("Expected %d overlap, found %d", largestCohort.CohortSize, len(subjectIds))
	}

	// now add a pair that overlaps with largestCohort:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    smallestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
		{
			CohortId1:    extendedCopyOfSecondLargestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
	}
	subjectIds = []*SubjectId{}
	population = largestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect overlap the size of the largestCohort-5:
	if len(subjectIds) != (largestCohort.CohortSize - 5) {
		t.Errorf("Expected %d overlap, found %d", largestCohort.CohortSize-5, len(subjectIds))
	}

	// order doesn't matter:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    extendedCopyOfSecondLargestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
		{
			CohortId1:    smallestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
	}
	subjectIds = []*SubjectId{}
	population = largestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect same as previous test above:
	if len(subjectIds) != (largestCohort.CohortSize - 5) {
		t.Errorf("Expected %d overlap, found %d", largestCohort.CohortSize-5, len(subjectIds))
	}

	// now test with two other cohorts that overlap:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    secondLargestCohort.Id,
			CohortId2:    extendedCopyOfSecondLargestCohort.Id,
			ProvidedName: "test"},
	}
	subjectIds = []*SubjectId{}
	population = extendedCopyOfSecondLargestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect overlap the size of the extendedCopyOfSecondLargestCohort.CohortSize - secondLargestCohort.CohortSize:
	if len(subjectIds) != (extendedCopyOfSecondLargestCohort.CohortSize - secondLargestCohort.CohortSize) {
		t.Errorf("Expected %d overlap, found %d", extendedCopyOfSecondLargestCohort.CohortSize-secondLargestCohort.CohortSize, len(subjectIds))
	}

	// now add in the largestCohort as a pair of extendedCopyOfSecondLargestCohort to the mix above:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    secondLargestCohort.Id,
			CohortId2:    extendedCopyOfSecondLargestCohort.Id,
			ProvidedName: "test"},
		{
			CohortId1:    largestCohort.Id,
			CohortId2:    extendedCopyOfSecondLargestCohort.Id,
			ProvidedName: "test"},
	}
	subjectIds = []*SubjectId{}
	population = extendedCopyOfSecondLargestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect overlap the size to be 0, since all items remaining from first pair happen to overlap with largestCohort and are therefore excluded (pair overlap is excluded):
	if len(subjectIds) != 0 {
		t.Errorf("Expected 0 overlap, found %d", len(subjectIds))
	}

	// now if the population is largestCohort, for the same pairs above, we expect the overlap to be 0 as well, as the first pair restricts the set for every other following pair (i.e. attrition at work):
	subjectIds = []*SubjectId{}
	population = largestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect overlap the size to be 0 as explained in comment above:
	if len(subjectIds) != 0 {
		t.Errorf("Expected 0 overlap, found %d", len(subjectIds))
	}

	// should return all in cohort:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{}
	subjectIds = []*SubjectId{}
	population = largestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect overlap the size to be the size of the cohort, since there are no filtering pairs:
	if len(subjectIds) != largestCohort.CohortSize {
		t.Errorf("Expected 0 overlap, found %d", len(subjectIds))
	}

	// should return 0:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    largestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
	}
	subjectIds = []*SubjectId{}
	population = largestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect overlap the size to be 0, since the pair is composed of the same cohort in CohortId1 and CohortId2 and their overlap is excluded:
	if len(subjectIds) != 0 {
		t.Errorf("Expected 0 overlap, found %d", len(subjectIds))
	}

	// should return 0:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    thirdLargestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
	}
	subjectIds = []*SubjectId{}
	population = smallestCohort
	resultsDataSource = tests.GetResultsDataSource()
	query = models.QueryFilterByCohortPairsHelper(filterCohortPairs, resultsDataSource, population.Id, "unionAndIntersect").
		Select("subject_id")
	_ = query.Scan(&subjectIds)
	// in this case we expect overlap the size to be 0, since the cohorts in the pair do not overlap with the population:
	if len(subjectIds) != 0 {
		t.Errorf("Expected 0 overlap, found %d", len(subjectIds))
	}
}

func TestRetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndTwoCohortPairsWithResults(t *testing.T) {
	setUp(t)
	filterIds := []int64{hareConceptId}
	populationCohort := largestCohort
	// setting the largest and smallest cohorts here as a pair:
	filterCohortPairs := []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    smallestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
	}
	breakdownConceptId := hareConceptId // not normally the case...but we'll use the same here just for the test...
	stats, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(testSourceId,
		populationCohort.Id, filterIds, filterCohortPairs, breakdownConceptId)
	// we expect results, and we expect the total of persons to be 6, since only 6 of the persons
	// in largestCohort have a HARE value (and smallestCohort does not overlap with largest):
	countPersons := 0
	for _, stat := range stats {
		countPersons += stat.NpersonsInCohortWithValue
	}
	if countPersons != 6 {
		t.Errorf("Expected 6 persons, found %d", countPersons)
	}
	// now if we add another cohort pair with secondLargestCohort and extendedCopyOfSecondLargestCohort,
	// then we should expect a reduction in the number of persons found. The reduction in this case
	// will take place because of a smaller intersection of the new cohorts with the population cohort,
	// and because of an overlaping person found in the two cohorts of the new pair.
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    smallestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
		{
			CohortId1:    secondLargestCohort.Id,
			CohortId2:    extendedCopyOfSecondLargestCohort.Id,
			ProvidedName: "test2"},
	}
	stats, _ = conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(testSourceId,
		populationCohort.Id, filterIds, filterCohortPairs, breakdownConceptId)
	countPersons = 0
	for _, stat := range stats {
		countPersons += stat.NpersonsInCohortWithValue
	}
	if countPersons != 4 {
		t.Errorf("Expected 4 persons, found %d", countPersons)
	}
}

func TestRetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairsWithResults(t *testing.T) {
	setUp(t)
	filterIds := []int64{hareConceptId}
	// setting the same cohort id here (artificial...but just to check if that returns the same value as when this filter is not there):
	filterCohortPairs := []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    secondLargestCohort.Id,
			CohortId2:    extendedCopyOfSecondLargestCohort.Id,
			ProvidedName: "test"},
	}
	breakdownConceptId := hareConceptId // not normally the case...but we'll use the same here just for the test...
	stats, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(testSourceId,
		extendedCopyOfSecondLargestCohort.Id, filterIds, filterCohortPairs, breakdownConceptId)
	// we expect values since secondLargestCohort has multiple subjects with hare info:
	if len(stats) < 4 {
		t.Errorf("Expected at least 4 results, found %d", len(stats))
	}
	prevName := ""
	for _, stat := range stats {
		// some very basic checks, making sure fields are not empty, repeated in next row, etc:
		if len(stat.ConceptValue) == len(stat.ValueName) ||
			len(stat.ConceptValue) == 0 ||
			len(stat.ValueName) == 0 ||
			stat.ValueAsConceptId == 0 ||
			stat.ValueName == prevName {
			t.Errorf("Invalid results")
		}
		prevName = stat.ValueName
	}
	// test without the filterCohortPairs, should return the same result:
	filterCohortPairs = []utils.CustomDichotomousVariableDef{}
	stats2, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(testSourceId,
		extendedCopyOfSecondLargestCohort.Id, filterIds, filterCohortPairs, breakdownConceptId)
	// very rough check (ideally we would check the individual stats as well...TODO?):
	if len(stats) > len(stats2) {
		t.Errorf("First query is more restrictive, so its stats should not be larger than stats2 of second query. Got %d and %d", len(stats), len(stats2))
	}
	// test filtering with smallest cohort, lenght should be 1, since that's the size of the smallest cohort:
	// setting the same cohort id here (artificial...normally it should be two different ids):
	filterCohortPairs = []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    smallestCohort.Id,
			CohortId2:    largestCohort.Id,
			ProvidedName: "test"},
	}
	stats3, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(testSourceId,
		secondLargestCohort.Id, filterIds, filterCohortPairs, breakdownConceptId)
	if len(stats3) != 2 {
		t.Errorf("Expected only two items in resultset, found %d", len(stats))
	}
}

func TestRetrieveBreakdownStatsBySourceIdAndCohortIdWithResults(t *testing.T) {
	setUp(t)
	breakdownConceptId := hareConceptId
	stats, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortId(testSourceId,
		secondLargestCohort.Id,
		breakdownConceptId)
	// we expect 5-1 rows since the largest test cohort has all HARE values represented in its population, but has NULL in the "OTH" entry:
	if len(stats) != 4 {
		t.Errorf("Expected 4 results, found %d", len(stats))
	}
}

// Tests what happens when persons have more than 1 HARE. This is a "data error" and should not
// happen in practice. The ideal solution would be for cohort-middleware to throw an error
// when it detects such a situation in the RetrieveBreakdownStats methods. This test shows that
// the code does not "hide" the error but instead returns the extra hare as an
// extra count, making the cohort numbers inconsistent and hopefully making the "data error" easy
// to spot.
// TODO - adjust the code to detect the issue and return an error, ideally with minimized or no repetition
// of the heavy queries in the RetrieveBreakdownStats methods... Idea: run this check as a QC query for each cohort
// at startup and write an ERROR to the log (with cohort id and name information) if it detects such data issues.
func TestRetrieveBreakdownStatsBySourceIdAndCohortIdWithResultsWithOnePersonTwoHare(t *testing.T) {
	setUp(t)
	breakdownConceptId := hareConceptId
	statsthirdLargestCohort, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortId(testSourceId,
		thirdLargestCohort.Id,
		breakdownConceptId)

	totalPersonInthirdLargestCohortWithValue := 0

	for _, statSecondLargest := range statsthirdLargestCohort {
		totalPersonInthirdLargestCohortWithValue += statSecondLargest.NpersonsInCohortWithValue
	}

	if totalPersonInthirdLargestCohortWithValue != thirdLargestCohort.CohortSize+1 {
		t.Errorf("Expected total peope in return data to be 1 larger than cohort size, but total people was %d and cohort size is %d", totalPersonInthirdLargestCohortWithValue, thirdLargestCohort.CohortSize)
	}

	statssecondLargestCohort, _ := conceptModel.RetrieveBreakdownStatsBySourceIdAndCohortId(testSourceId,
		secondLargestCohort.Id,
		breakdownConceptId)

	totalPersonInsecondLargestCohortWithValue := 0

	for _, statLargeCohort := range statssecondLargestCohort {
		totalPersonInsecondLargestCohortWithValue += statLargeCohort.NpersonsInCohortWithValue
	}

	expectedWithValueInSecondLargest := secondLargestCohort.CohortSize - 1 // because 2nd largest has one person that has a NULL HARE entry...
	if totalPersonInsecondLargestCohortWithValue != expectedWithValueInSecondLargest+1 {
		t.Errorf("Expected total peope in return data to be 1 larger than nr distinct persons with HARE, but total was %d and nr distinct persons with HARE +1 is %d", totalPersonInsecondLargestCohortWithValue, expectedWithValueInSecondLargest+1)
	}
}

func TestGetAllCohortDefinitionsAndStatsOrderBySizeDesc(t *testing.T) {
	setUp(t)
	cohortDefinitions, _ := cohortDefinitionModel.GetAllCohortDefinitionsAndStatsOrderBySizeDesc(testSourceId)
	if len(cohortDefinitions) != len(allCohortDefinitions) {
		t.Errorf("Found %d", len(cohortDefinitions))
	}
	// check if stats fields are filled and if order is as expected:
	previousSize := 1000000
	for _, cohortDefinition := range cohortDefinitions {
		if cohortDefinition.CohortSize <= 0 {
			t.Errorf("Expected positive value, found %d", cohortDefinition.CohortSize)
		}
		if cohortDefinition.CohortSize > previousSize {
			t.Errorf("Data not ordered by size descending!")
		}
		previousSize = cohortDefinition.CohortSize
	}
}

func TestGetCohortName(t *testing.T) {
	setUp(t)
	allCohortDefinitions, _ := cohortDefinitionModel.GetAllCohortDefinitions()
	firstCohortId := allCohortDefinitions[0].Id
	cohortName, _ := cohortDefinitionModel.GetCohortName(firstCohortId)
	if cohortName != allCohortDefinitions[0].Name {
		t.Errorf("Expected %s", allCohortDefinitions[0].Name)
	}
}

func TestGetCohortDefinitionByName(t *testing.T) {
	setUp(t)
	cohortDefinition, _ := cohortDefinitionModel.GetCohortDefinitionByName(smallestCohort.Name)
	if cohortDefinition == nil || cohortDefinition.Name != smallestCohort.Name {
		t.Errorf("Expected %s", smallestCohort.Name)
	}
}

func TestRetrieveHistogramDataBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(t *testing.T) {
	setUp(t)
	filterConceptIds := []int64{}
	filterCohortIds := []utils.CustomDichotomousVariableDef{}
	data, _ := cohortDataModel.RetrieveHistogramDataBySourceIdAndCohortIdAndConceptIdsAndCohortPairs(testSourceId, largestCohort.Id, histogramConceptId, filterConceptIds, filterCohortIds)
	if len(data) == 0 {
		t.Errorf("expected 1 or more histogram data but got 0")
	}
}

func TestQueryFilterByConceptIdsAndCohortPairsHelper(t *testing.T) {
	// This test checks whether the query succeeds when the mainObservationTableAlias
	// argument passed to QueryFilterByConceptIdsAndCohortPairsHelper (last argument)
	// matches the alias used in the main query, and whether it fails otherwise.

	setUp(t)
	omopDataSource := tests.GetOmopDataSource()
	filterConceptIds := []int64{allConceptIds[0], allConceptIds[1], allConceptIds[2]}
	filterCohortPairs := []utils.CustomDichotomousVariableDef{} // empty / not really needed for test
	var personIds []struct {
		PersonId int64
	}

	// Subtest1: correct alias "observation":
	query := omopDataSource.Db.Table(omopDataSource.Schema + ".observation_continuous as observation" + omopDataSource.GetViewDirective()).
		Select("observation.person_id")
	query = models.QueryFilterByConceptIdsAndCohortPairsHelper(query, testSourceId, filterConceptIds, filterCohortPairs, omopDataSource, "", "observation")
	meta_result := query.Scan(&personIds)
	if meta_result.Error != nil {
		t.Errorf("Did NOT expect an error")
	}
	// Subtest2: incorrect alias "observation"...should fail:
	query = omopDataSource.Db.Table(omopDataSource.Schema + ".observation_continuous as observationWRONG").
		Select("*")
	query = models.QueryFilterByConceptIdsAndCohortPairsHelper(query, testSourceId, filterConceptIds, filterCohortPairs, omopDataSource, "", "observation")
	meta_result = query.Scan(&personIds)
	if meta_result.Error == nil {
		t.Errorf("Expected an error")
	}
}

func TestRetrieveDataBySourceIdAndCohortIdAndConceptIdsOrderedByPersonId(t *testing.T) {
	setUp(t)
	cohortDefinitions, _ := cohortDefinitionModel.GetAllCohortDefinitionsAndStatsOrderBySizeDesc(testSourceId)
	var sumNumeric float32 = 0
	textConcat := ""
	for _, cohortDefinition := range cohortDefinitions {

		cohortData, _ := cohortDataModel.RetrieveDataBySourceIdAndCohortIdAndConceptIdsOrderedByPersonId(
			testSourceId, cohortDefinition.Id, allConceptIds)

		// 1- cohortData items > 0, assuming each cohort has a person wit at least one observation
		if len(cohortData) <= 0 {
			t.Errorf("Expected some cohort data")
		}
		var previousPersonId int64 = -1
		for _, cohortDatum := range cohortData {
			// check for order: person_id is not smaller than previous person_id
			if cohortDatum.PersonId < previousPersonId {
				t.Errorf("Data not ordered by person_id!")
			}
			previousPersonId = cohortDatum.PersonId
			sumNumeric += cohortDatum.ConceptValueAsNumber
			textConcat += cohortDatum.ConceptValueAsString
		}
	}
	// check for data: sum of all numeric values > 0
	if sumNumeric == 0 {
		t.Errorf("Expected some numeric cohort data")
	}
	// check for data: concat of all string values != ""
	if textConcat == "" {
		t.Errorf("Expected some string cohort data")
	}
}

func TestErrorForRetrieveDataBySourceIdAndCohortIdAndConceptIdsOrderedByPersonId(t *testing.T) {
	// Tests if the method returns an error when query fails.

	cohortDefinitions, _ := cohortDefinitionModel.GetAllCohortDefinitionsAndStatsOrderBySizeDesc(testSourceId)

	// break something in the Results schema to cause a query failure in the next method:
	tests.BreakSomething(models.Results, "cohort", "cohort_definition_id")
	// set last action to restore back:
	// run test:
	_, error := cohortDataModel.RetrieveDataBySourceIdAndCohortIdAndConceptIdsOrderedByPersonId(
		testSourceId, cohortDefinitions[0].Id, allConceptIds)
	if error == nil {
		t.Errorf("Expected error")
	}
	// revert the broken part:
	tests.FixSomething(models.Results, "cohort", "cohort_definition_id")
}

// for given source and cohort, counts how many persons have the given HARE value
func getNrPersonsWithHareConceptValue(sourceId int, cohortId int, hareConceptValue int64) int64 {
	conceptIds := []int64{hareConceptId}
	personLevelData, _ := cohortDataModel.RetrieveDataBySourceIdAndCohortIdAndConceptIdsOrderedByPersonId(sourceId, cohortId, conceptIds)
	var count int64 = 0
	for _, personLevelDatum := range personLevelData {
		if personLevelDatum.ConceptValueAsConceptId == hareConceptValue {
			count++
		}
	}
	return count
}

func TestRetrieveCohortOverlapStats(t *testing.T) {
	// Tests if we get the expected overlap
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := secondLargestCohort.Id // to ensure we get some overlap, just repeat the same here...
	filterConceptId := hareConceptId
	filterConceptValue := asnHareConceptId
	otherFilterConceptIds := []int64{}
	filterCohortPairs := []utils.CustomDichotomousVariableDef{}
	stats, _ := cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
	// get the number of persons in this cohort that have this filterConceptValue:
	nr_expected := getNrPersonsWithHareConceptValue(testSourceId, caseCohortId, filterConceptValue)
	if nr_expected == 0 {
		t.Errorf("Expected nr persons with HARE value should be > 0")
	}
	if stats.CaseControlOverlap != nr_expected {
		t.Errorf("Expected overlap of %d, but found %d", nr_expected, stats.CaseControlOverlap)
	}
}

func TestRetrieveCohortOverlapStatsScenario2(t *testing.T) {
	// Tests if we get the expected overlap
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := secondLargestCohort.Id // to ensure we get some overlap, just repeat the same here...
	filterConceptId := hareConceptId
	filterConceptValue := asnHareConceptId
	otherFilterConceptIds := []int64{hareConceptId} // repeat hare concept id here...Artificial, but will ensure overlap
	filterCohortPairs := []utils.CustomDichotomousVariableDef{}
	stats, _ := cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
	// get the number of persons in this cohort that have this filterConceptValue:
	nr_expected := getNrPersonsWithHareConceptValue(testSourceId, caseCohortId, filterConceptValue)
	if nr_expected == 0 {
		t.Errorf("Expected nr persons with HARE value should be > 0")
	}
	if stats.CaseControlOverlap != nr_expected {
		t.Errorf("Expected overlap of %d, but found %d", nr_expected, stats.CaseControlOverlap)
	}
}

func TestRetrieveCohortOverlapStatsScenario3(t *testing.T) {
	// Tests if we get the expected overlap
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := secondLargestCohort.Id // to ensure we get some overlap, just repeat the same here...
	filterConceptId := hareConceptId
	filterConceptValue := asnHareConceptId // filter on 'ASN'
	otherFilterConceptIds := []int64{}
	filterCohortPairs := []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    extendedCopyOfSecondLargestCohort.Id,
			CohortId2:    smallestCohort.Id,
			ProvidedName: "test",
		},
	}
	stats, _ := cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
	// there are 2 persons with 'ASN' value for HARE in secondLargestCohort, so expect 2:
	if stats.CaseControlOverlap != 2 {
		t.Errorf("Expected overlap of 2, but found %d", stats.CaseControlOverlap)
	}
}

func TestRetrieveCohortOverlapStatsZeroOverlap(t *testing.T) {
	// Tests if a scenario where NO overlap is expected indeed results in 0
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := smallestCohort.Id
	filterConceptId := hareConceptId
	var filterConceptValue int64 = -1 // should result in 0 overlap
	otherFilterConceptIds := []int64{}
	filterCohortPairs := []utils.CustomDichotomousVariableDef{}
	stats, _ := cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
	if stats.CaseControlOverlap != 0 {
		t.Errorf("Expected overlap of 0, but found %d", stats.CaseControlOverlap)
	}
}

func TestRetrieveCohortOverlapStatsZeroOverlapScenario2(t *testing.T) {
	// Tests if a scenario where NO overlap ends in the expected error/panic
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := secondLargestCohort.Id // to ensure THIS part does not cause the 0 overlap, just repeat the same...
	filterConceptId := hareConceptId
	filterConceptValue := asnHareConceptId
	// set this list to some dummy non-existing ids:
	otherFilterConceptIds := []int64{-1, -2}
	filterCohortPairs := []utils.CustomDichotomousVariableDef{}

	// the RetrieveCohortOverlapStats below should result in panic/error:
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	_, _ = cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
}

func TestRetrieveCohortOverlapStatsZeroOverlapScenario3(t *testing.T) {
	// Tests if a scenario where NO overlap is expected indeed results in 0
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := secondLargestCohort.Id // to ensure THIS part does not cause the 0 overlap, just repeat the same...
	filterConceptId := hareConceptId
	filterConceptValue := asnHareConceptId // filter on 'ASN'
	otherFilterConceptIds := []int64{}
	filterCohortPairs := []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    extendedCopyOfSecondLargestCohort.Id, // does not really matter which cohort here, as long as CohortId1 and CohortId2 are the same it should result in an empty set since we remove the intersecting part of the cohorts in a dichotomous pair
			CohortId2:    extendedCopyOfSecondLargestCohort.Id,
			ProvidedName: "test",
		},
	}
	stats, _ := cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
	if stats.CaseControlOverlap != 0 {
		t.Errorf("Expected overlap of 0, but found %d", stats.CaseControlOverlap)
	}
}

func TestRetrieveCohortOverlapStatsWithCohortPairs(t *testing.T) {
	// Tests if we get the expected overlap
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := secondLargestCohort.Id // to ensure we get overlap, just repeat the same here...
	filterConceptId := hareConceptId
	filterConceptValue := asnHareConceptId          // the cohorts we use below both have persons with "ASN" HARE value
	otherFilterConceptIds := []int64{hareConceptId} // repeat hare concept id here...Artificial, but will ensure overlap
	filterCohortPairs := []utils.CustomDichotomousVariableDef{
		{
			CohortId1:    smallestCohort.Id,
			CohortId2:    thirdLargestCohort.Id,
			ProvidedName: "test"}, // pair1
		{
			CohortId1:    thirdLargestCohort.Id,
			CohortId2:    smallestCohort.Id,
			ProvidedName: "test"}, // pair2 (same as above, but switched...artificial, but will ensure some data):
	}
	stats, _ := cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
	// get the number of persons in the smaller cohorts that have this filterConceptValue (this can be the expected nr because
	// the secondLargestCohort in this case contains all other cohorts):
	nr_expected := getNrPersonsWithHareConceptValue(testSourceId, thirdLargestCohort.Id, filterConceptValue)
	nr_expected = nr_expected + getNrPersonsWithHareConceptValue(testSourceId, smallestCohort.Id, filterConceptValue)
	if nr_expected == 0 {
		t.Errorf("Expected nr persons with HARE value should be > 0")
	}
	if stats.CaseControlOverlap != nr_expected {
		t.Errorf("Expected overlap of %d, but found %d", nr_expected, stats.CaseControlOverlap)
	}
	filterCohortPairs = []utils.CustomDichotomousVariableDef{}
	// without the restrictive filter on cohort pairs, the result should be bigger, as the largest cohort has more persons with
	// the asnHareConceptId than the ones used in the pairs above:
	stats2, _ := cohortDataModel.RetrieveCohortOverlapStats(testSourceId, caseCohortId, controlCohortId,
		filterConceptId, filterConceptValue, otherFilterConceptIds, filterCohortPairs)
	if stats.CaseControlOverlap >= stats2.CaseControlOverlap {
		t.Errorf("Expected overlap in first query to be smaller than in second one")
	}
}

func TestRetrieveCohortOverlapStatsWithoutFilteringOnConceptValue(t *testing.T) {
	// Tests if we get the expected overlap
	setUp(t)
	caseCohortId := secondLargestCohort.Id
	controlCohortId := secondLargestCohort.Id // to ensure we get some overlap, just repeat the same here...
	otherFilterConceptIds := []int64{}
	filterCohortPairs := []utils.CustomDichotomousVariableDef{}
	stats, _ := cohortDataModel.RetrieveCohortOverlapStatsWithoutFilteringOnConceptValue(testSourceId, caseCohortId, controlCohortId,
		otherFilterConceptIds, filterCohortPairs)
	// basic test:
	if stats.CaseControlOverlap == 0 {
		t.Errorf("Expected nr persons to be > 0")
	}
}

func TestValidateObservationData(t *testing.T) {
	// Tests if we get the expected validation results
	setUp(t)
	var cohortDataModel = new(models.CohortData)
	nrIssues, error := cohortDataModel.ValidateObservationData([]int64{hareConceptId})
	// we know that the test dataset has at least one patient with more than one HARE:
	if error != nil {
		t.Errorf("Did not expect an error, but got %v", error)
	}
	if nrIssues == 0 {
		t.Errorf("Expected validation issues")
	}
	nrIssues2, error := cohortDataModel.ValidateObservationData([]int64{456789999}) // some random concept id not in db
	// we expect no results for a concept that does not exist:
	if error != nil {
		t.Errorf("Did not expect an error, but got %v", error)
	}
	if nrIssues2 != 0 {
		t.Errorf("Expected NO validation issues")
	}
	nrIssues3, error := cohortDataModel.ValidateObservationData([]int64{})
	// we expect no results for an empty concept list:
	if error != nil {
		t.Errorf("Did not expect an error, but got %v", error)
	}
	if nrIssues3 != -1 {
		t.Errorf("Expected result to be -1")
	}
}

func TestGetVersion(t *testing.T) {
	// mock values (in reality these are set at build time - see Dockerfile "go build" "-ldflags" argument):
	version.GitCommit = "abc"
	version.GitVersion = "def"
	v := versionModel.GetVersion()
	if v.GitCommit != version.GitCommit || v.GitVersion != version.GitVersion {
		t.Errorf("Wrong value")
	}
}

func TestGetSourceByName(t *testing.T) {
	allSources, _ := sourceModel.GetAllSources()
	foundSource, _ := sourceModel.GetSourceByName(allSources[0].SourceName)
	if allSources[0].SourceName != foundSource.SourceName {
		t.Errorf("Expected data not found")
	}
}

func TestGetSourceById(t *testing.T) {
	allSources, _ := sourceModel.GetAllSources()
	foundSource, _ := sourceModel.GetSourceById(allSources[0].SourceId)
	if allSources[0].SourceId != foundSource.SourceId {
		t.Errorf("Expected data not found")
	}
}

func TestGetCohortDefinitionById(t *testing.T) {
	allCohortDefinitions, _ := cohortDefinitionModel.GetAllCohortDefinitions()
	foundCohortDefinition, _ := cohortDefinitionModel.GetCohortDefinitionById(allCohortDefinitions[0].Id)
	if allCohortDefinitions[0].Id != foundCohortDefinition.Id {
		t.Errorf("Expected data not found")
	}
}

func TestRetrieveDataByOriginalCohortAndNewCohort(t *testing.T) {
	setUp(t)
	originalCohortSize := thirdLargestCohort.CohortSize
	originalCohortId := thirdLargestCohort.Id
	cohortDefinitionId := secondLargestCohort.Id

	personIdAndCohortList, _ := cohortDataModel.RetrieveDataByOriginalCohortAndNewCohort(testSourceId, originalCohortId, cohortDefinitionId)
	if len(personIdAndCohortList) != originalCohortSize {
		t.Errorf("length of return data does not match number of people in cohort")
	}

	for _, personIdAndCohort := range personIdAndCohortList {
		if personIdAndCohort.CohortId != int64(cohortDefinitionId) {
			t.Errorf("cohort_id we retireved is not correct")
		}
		if personIdAndCohort.PersonId == int64(0) {
			t.Error("person id should be valid and not 0")
		}
	}
}
