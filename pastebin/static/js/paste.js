'use strict';

(function () {
  window.addEventListener('DOMContentLoaded', () => {
    // Fancy delete button
    document.getElementById('deleteform').addEventListener('submit', function (e) {
      e.preventDefault();

      document.getElementById('delete-btn').setAttribute('disabled', true);
      let data = new FormData(this);
      fetch(this.getAttribute('action'), {
        body: data,
        headers: {
          'X-Requested-With': 'XMLHttpRequest',
        },
        method: this.getAttribute('method'),
      }).then(response => {
        if (response.ok) {
          return response.text();
        } else {
          alert(`Oops, we couldn't delete this paste :( The following was encountered:\n\n${response.status}: ${response.statusText}`);
          throw "-flails-";
        }
      }).then(result => {
        alert("BAM!@ Okay! This paste is no longer available.");
        location.replace(result);
      }).catch(error => {
        if (error != "-flails-") {
          alert("Oops, we couldn't delete your paste :( Maybe the network pipes aren't up?");
          document.getElementById('delete-btn').removeAttribute('disabled');
        }
      });
    });

    // Iframe auto-resize
    document.getElementById('content-frame').addEventListener('load', function () {
      try {
        let height = this.contentDocument.body.scrollHeight;
        this.style.height = height + 24 + "px";
      } catch { };
    });

    // Fancy content fetch
    let fetchContent = function(contentURL) {
      fetch(contentURL).then(response => {
        if (response.ok) {
          if (document.getElementById('format').value == 'html') {
            return response.blob();
          } else {
            return response.text();
          }
        } else {
          document.getElementById('loader-result').textContent = `Meep! I couldn't get the content -flails- (${response.status}: ${response.statusText})`;
          throw "-flails-";
        }
      }).then(result => {
        if (document.getElementById('format').value == 'html') {
          let blob = new Blob([result], { type: 'text/html' });
          let url = URL.createObjectURL(blob);
          document.getElementById('content-frame').src = url;
          document.getElementById('content-frame').classList.remove('d-none');
          document.getElementById('loader').classList.add('d-none');
        } else {
          document.getElementById('content-text').classList.remove('d-none');
          // document.getElementById('content-text').textContent = result;
          document.getElementById('content-text').innerHTML = result.replace(/^(.*)$/mg, "<span class=\"line\">$1</span>")

          document.getElementById('loader').classList.add('d-none');
        }
      }).catch(error => {
        if (error != "-flails-") {
          document.getElementById('loader-result').textContent = "Meep! I couldn't get your content :( Maybe the network pipes aren't up?";
        }

        document.getElementById('loader').classList.remove('text-light');
        document.getElementById('loader').classList.add('text-danger');
      });
    }

    document.getElementById('loader').classList.remove('d-none');
    let pasteID = document.getElementById('pasteID').value;
    let contentURL = "/pastebinc/" + pasteID + "/content";
    if (document.querySelectorAll('#driveHosted').length > 0) {
      fetch("/pastebinc/" + pasteID + "/content/link").then(response => {
        if (response.ok) {
          return response.text();
        } else {
          document.getElementById('loader-result').textContent = `Meep! I couldn't get the content link -flails- (${response.status}: ${response.statusText})`;
          throw "-flails-";
        }
      }).then(result => {
        fetchContent(result);
      }).catch(error => {
        if (error != "-flails-") {
          document.getElementById('loader-result').textContent = "Meep! I couldn't get your content :( Maybe the network pipes aren't up?";
        }

        document.getElementById('loader').classList.remove('text-light');
        document.getElementById('loader').classList.add('text-danger');
      });
    } else {
      if (document.getElementById('format').value == 'html') {
        document.getElementById('content-frame').src = contentURL;
        document.getElementById('content-frame').classList.remove('d-none');
        document.getElementById('loader').classList.add('d-none');
        return;
      } else {
        fetchContent(contentURL);
      }
    }

    grecaptcha.ready(function() {
      let rkey = document.getElementById('recaptcha-key').value;
      grecaptcha.execute(rkey, {action: 'cpaste'});
    });
  });
})();
