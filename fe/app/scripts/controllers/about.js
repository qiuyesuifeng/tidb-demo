'use strict';

/**
 * @ngdoc function
 * @name feApp.controller:AboutCtrl
 * @description
 * # AboutCtrl
 * Controller of the feApp
 */
angular.module('feApp')
  .controller('AboutCtrl', function ($scope) {
    $scope.awesomeThings = [
      'HTML5 Boilerplate',
      'AngularJS',
      'Karma'
    ];
  });
