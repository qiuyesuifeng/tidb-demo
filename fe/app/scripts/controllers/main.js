'use strict';

/**
 * @ngdoc function
 * @name feApp.controller:MainCtrl
 * @description
 * # MainCtrl
 * Controller of the feApp
 */
angular.module('feApp')
  .controller('MainCtrl',['$scope','$http','$interval', function ($scope, $http, $interval) {
        $scope.count = 0;
        $interval(function(){
            $http.get("http://127.0.0.1:10081/api/v1/counter").  
            success(function(res) 
                {
                    console.log(res);
                    $scope.count = res.count;
                })
               .error(function(status) 
               { 
               });
        }, 1000);
  }]);
