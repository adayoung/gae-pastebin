'use strict';

window.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.tagbox').forEach(element => {
    element.addEventListener('input', function () {
      this.value = this.value.toLowerCase().replace(RegExp('[^a-z0-9 ]+', 'g'), '', 'g');
      this.value = this.value.replace(RegExp('[ ]+', 'g'), ' ', 'g');

      let tags = this.value.split(' ');
      tags = Array.from(new Set(tags)); // remove duplicates
      tags = tags.slice(0, 15); // tags max limit is 15 tags
      tags = tags.map(e => { // tag max length is 15 characters
        if (e.length > 15) {
          e = e.substr(0, 15);
        }
        return e
      });

      this.value = tags.join(' ');
    });
  });
});
