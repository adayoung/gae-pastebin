$(document).ready(function(){
  $('.nojs').hide();
  $('.havejs').show();

  $('#label_plain').addClass('active');
  $('#plain').attr('checked', true);
  $('#spinner').attr('src', $('#spinner').data('src'));
  $('#appengine').attr('src', $('#appengine').data('src'));

  $('#paste_btn').on('click', function(event){
    event.preventDefault();

    if (!$('#content').val().length > 0) {
      return false;
    }

    $('#paste_btn').text('Please wait...');
    $('#paste_btn').addClass('disabled');

    $('#searchbox').attr('disabled', true);
    $('#content').attr('disabled', true);
    $('#title').attr('disabled', true);
    $('#tags').attr('disabled', true);

    $('#spinner').toggle(true);
    $.post(location.href, {
      content: $('#content').val(),
      title: $('#title').val(),
      tags: $('#tags').val(),
      format: $('input[name=format]:checked').val(),
      "gorilla.csrf.Token": $('input[name="gorilla.csrf.Token"]').val()
    }).done(function(e){
      location.replace(e);
    }).fail(function(e){
      alert("Oops, we couldn't post your paste :( The following was encountered:\n\n" + e.status + " - " + e.statusText);
      $('#searchbox').attr('disabled', false);
      $('#content').attr('disabled', false);
      $('#title').attr('disabled', false);
      $('#tags').attr('disabled', false);
      $('#spinner').toggle(false);

      $('#paste_btn').text('Paste it!');
      $('#paste_btn').removeClass('disabled');
      $('#content').focus();
      $('#content').select();
    });

  });

  $('#content').bind('input propertychange', function(){
    var noc = $('#content').val().length;
    $('#noc').text(noc);

    if (noc > 950 * 1024) {
      $('#eep').toggle(true);
      $('#paste_btn').text('Paste it anyway!');
      $('#paste_btn').addClass('btn-danger');
      $('#noc_wrap').addClass('text-danger');
      // $('#paste_btn').addClass('disabled');
    } else {
      $('#paste_btn').text('Paste it!');
      $('#paste_btn').removeClass('btn-danger');
      $('#eep').toggle(false);
      // $('#paste_btn').removeClass('disabled');
      $('#noc_wrap').removeClass('text-danger');
    }
  });

  $(document).on('keydown', function(e){
    if (e.ctrlKey && e.keyCode == 13) {
      $('#plain').click();
      $('#paste_btn').click();
    }

    if (e.altKey && e.keyCode == 13) {
      $('#html').click();
      $('#paste_btn').click();
    }
  });

  $('.noenter').on('keypress', function(e){
    if (e.which == 13) {
      return false;
    }
  });

  $('#paste_btn').tooltip({
    placement: 'auto',
    title: $('#paste_btn').data('title')
  });

  $('#content').focus();
});