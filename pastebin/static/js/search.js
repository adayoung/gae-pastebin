'use strict';

(function () {
  window.addEventListener('DOMContentLoaded', () => {
    // Javascript enabled features
    document.getElementById('results').classList.remove('d-none');
    document.getElementById('loadmore').classList.remove('d-none');

    document.getElementById('loadmore').addEventListener('click', function (e) {
      e.preventDefault();

      this.classList.add('disabled');
      this.textContent = "Please wait...";

      let url = new URL(location.href);
      url.searchParams.set('c', this.dataset.cursor);
      fetch(url, {
        headers: {
          'X-Requested-With': 'XMLHttpRequest',
        }
      }).then(response => {
        if (response.ok) {
          return response.json();
        } else {
          alert(`Oops, we couldn't get search results :( The following was encountered:\n\n${response.status}: ${response.statusText}`);
          throw "-flails-";
        }
      }).then(result => {
        this.dataset.cursor = result.cursor;
        if (result.paste.results == null) {
          this.textContent = "No more results ...";
          this.classList.add('d-none');
        } else {
          this.href = url.toString();
          this.textContent = "Load more results";
          this.classList.remove('disabled');

          result.paste.results.forEach(item => {
            let tags = [];
            item.tags.forEach(tag => {
              let colour = result.paste.tags.indexOf(tag) > -1 ? 'primary' : 'secondary';
              tags.push(`
                <a class="text-decoration-none" href="/pastebin/search/?tags=${tag}">
                  <span class="badge bg-${colour} me-1">${tag}</span>
                </a>
              `);
            });

            if (item.title.length == 0) {
              item.title = item.paste_id;
            }

            let row = `
              <tr>
                <td><a class="text-decoration-none" href="/pastebin/${item.paste_id}">${item.title}</a></td>
                <td title="${item.i_date}">${item.date}</td>
                <td>${tags.join('')}</td>
              </tr>
            `;
            document.getElementById('results-body').insertAdjacentHTML("beforeend", row);

            if (result.paste.results.length < 10) {
              this.classList.add('d-none');
            }
          });
        }
      }).catch(error => {
        if (error != "-flails-") {
          alert("Oops, we couldn't get search results :( Maybe the network pipes aren't up?");
        }

        this.textContent = "Load more results";
        this.classList.remove('disabled');
      });
    });

    document.getElementById('loadmore').click();
  });
})();
