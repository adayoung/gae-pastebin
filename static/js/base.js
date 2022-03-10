'use strict';

(function () {
  window.addEventListener('DOMContentLoaded', () => {
    let offset = 220;
    let rateLimit = false;

    document.addEventListener('scroll', () => {
      if (!rateLimit) {
        window.requestAnimationFrame(function() {
          if (window.scrollY > offset) {
            document.getElementById('back-to-top').classList.remove('d-none');
          } else {
            document.getElementById('back-to-top').classList.add('d-none');
          }
          rateLimit = false;
        }, {
          passive: true
        });

        rateLimit = true;
      }
    });
  });
})();

window.addEventListener("load", function () {
  window.cookieconsent.initialise({
    "palette": {
      "popup": {
        "background": "#edeff5",
        "text": "#838391"
      },
      "button": {
        "background": "#4b81e8"
      }
    },
    "theme": "edgeless",
    "content": {
      "href": "https://ada-young.com/pastebin/about#PrivacyPolicy"
    }
  })
});
