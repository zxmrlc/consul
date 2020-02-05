@setupApplicationTest
@notNamespaceable

Feature: notices / show: Show notices Page
  Scenario: I see the Notices page
    Given 1 datacenter model with the value "datacenter"
    When I visit the notices page
    Then the url should be /notices
    And the title should be "Notices - Consul"