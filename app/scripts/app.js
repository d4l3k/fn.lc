/*global window, document, addEventListener, Polymer, Trianglify*/

/*
Copyright (c) 2015 The Polymer Project Authors. All rights reserved.
This code may only be used under the BSD style license found at http://polymer.github.io/LICENSE.txt
The complete set of authors may be found at http://polymer.github.io/AUTHORS.txt
The complete set of contributors may be found at http://polymer.github.io/CONTRIBUTORS.txt
Code distributed by Google as part of the polymer project is also
subject to an additional IP rights grant found at http://polymer.github.io/PATENTS.txt
*/


(function(document) {
  'use strict';

  var app = document.querySelector('#app');

  app.displayInstalledToast = function() {
    document.querySelector('#caching-complete').show();
  };

  app.addEventListener('dom-change', function() {
    console.log('Our app is ready to rock!');
  });

  window.addEventListener('WebComponentsReady', function() {
    console.log('WebComponentsReady');
  });

  // Main area's paper-scroll-header-panel custom condensing transformation of
  // the appName in the middle-container and the bottom title in the bottom-container.
  // The appName is moved to top and shrunk on condensing. The bottom sub title
  // is shrunk to nothing on condensing.
  addEventListener('paper-header-transform', function(e) {
    var appName = document.querySelector('.app-name');
    var middleContainer = document.querySelector('.middle-container');
    var bottomContainer = document.querySelector('.bottom-container');
    var detail = e.detail;
    var heightDiff = detail.height - detail.condensedHeight;
    var yRatio = Math.min(1, detail.y / heightDiff);
    var maxMiddleScale = 0.50;  // appName max size when condensed. The smaller the number the smaller the condensed size.
    var scaleMiddle = Math.max(maxMiddleScale, (heightDiff - detail.y) / (heightDiff / (1-maxMiddleScale))  + maxMiddleScale);
    var scaleBottom = 1 - yRatio;

    // Move/translate middleContainer
    Polymer.Base.transform('translate3d(0,' + yRatio * 100 + '%,0)', middleContainer);

    // Scale bottomContainer and bottom sub title to nothing and back
    Polymer.Base.transform('scale(' + scaleBottom + ') translateZ(0)', bottomContainer);

    // Scale middleContainer appName
    Polymer.Base.transform('scale(' + scaleMiddle + ') translateZ(0)', appName);
  });

  // Close drawer after menu item is selected if drawerPanel is narrow
  app.onMenuSelect = function() {
    var drawerPanel = document.querySelector('#paperDrawerPanel');
    if (drawerPanel.narrow) {
      drawerPanel.closeDrawer();
    }
  };


  // Background
  var backgrounds = [
    document.getElementById('background'),
    document.getElementById('background2'),
  ];
  window.renderBackground = function(first) {
    var pattern = Trianglify({
        width: window.innerWidth,
        height: window.innerHeight*4,
    });
    if (first) {
      pattern.canvas(backgrounds[0]);
      return;
    }
    pattern.canvas(backgrounds[1]);
    backgrounds[1].classList.add('fade-in');
    setTimeout(function() {
      pattern.canvas(backgrounds[0]);
      backgrounds[1].classList.remove('fade-in');
    }, 250);
  };
  window.renderBackground(true);
  window.addEventListener('hashchange', function() {
    window.renderBackground();
  });
  window.addEventListener('WebComponentsReady', function() {
    document.querySelector('#mainContainer').addEventListener('scroll', function() {
      var translate = this.scrollTop/this.scrollHeight*window.innerHeight;
      backgrounds.forEach(function(background) {
        background.style.transform = 'translate3d(0,-'+translate+'px,0)';
      });
    });
  });
}(document));
