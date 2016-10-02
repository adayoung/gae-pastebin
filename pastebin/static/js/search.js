$(document).ready(function(){
  $('.havejs').show();

  $('#loadmore').on('click', function(e) {
    e.preventDefault();

    $(this).addClass('disabled');
    $(this).text('Please wait ...');

    $.getJSON(location.href + '&c=' + $(this).data('cursor')).done(function(data) {
      $('#loadmore').data('cursor', data.cursor);
      $('#loadmore').attr('href', location.href + '&c=' + data.cursor);

      if (data.paste.results == null) {
        $('#loadmore').text('No more results ...');
        $('#loadmore').fadeOut(1500);
        return false;
      }

      var lastindex = parseInt($('#results tbody tr').last().find('td').html()) + 1 || 1;

      for (var i = 0; i < data.paste.results.length; i++) {
        var tags = data.paste.results[i].tags;
        var htags = [];
        for (var j=0; j < tags.length; j++) {
          var label_class = data.paste.tags.indexOf(tags[j]) > -1 ? 'primary' : 'default';
          htags.push('<a class="label label-' + label_class + '" href="/pastebin/search/?tags=' + tags[j] + '">' + tags[j] + '</a>');
        }

        $('#results tbody').append('<tr class="ajaxload"><td>' + parseInt(lastindex + i)  +  '</td><td><a href="/pastebin/' + data.paste.results[i].paste_id  + '">' + data.paste.results[i].title  + '</a></td><td title="' + data.paste.results[i].i_date + '">' + data.paste.results[i].date + '</td><td>' + htags.join(' ') +  '</td></tr>');
      }

      $('#loadmore').removeClass('disabled');
      $('#loadmore').text('Load more results');
      $('.ajaxload').fadeIn();
    }).fail(function(e){
      // Huh? Why is it coming here?!
      alert("Oops, we couldn't get search results :( The following was encountered:\n\n" + e.status + " - " + e.statusText + '\n' + e.responseText);
    });
  });

  $('#loadmore').click();
});