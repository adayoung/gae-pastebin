var auth_window;
$(document).ready(function(){
  $('.nojs').hide();
  $('.havejs').show();
  $('.tehcontrols').css('display', 'inline-block')
  $('#paste_gdrv').attr('disabled', false);

  $('#plain').prop('checked', true);
  $('input[type=radio]:checked').parent().addClass('active');
  $('#spinner').attr('src', $('#spinner').data('src'));
  $('#appengine').attr('src', $('#appengine').data('src'));

  $('#about_btn').on('click', function(event){
    event.preventDefault();
    window.open($(this).attr('href'));
  });

  $('#paste_btn').on('click', function(event){
    event.preventDefault();

    if (!$('#content').val().length > 0) {
      $('#content').parent().addClass('has-error');
      $('#content').focus();
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
      destination: $('input[name=destination]:checked').val(),
      "gorilla.csrf.Token": $('input[name="gorilla.csrf.Token"]').val()
    }).done(function(e){
      location.replace(e);
    }).fail(function(e){
      alert("Oops, we couldn't post your paste :( The following was encountered:\n\n" + e.status + " - " + e.statusText + '\n' + e.responseText);
      HandlePasteError(e);
      $('#searchbox').attr('disabled', false);
      $('#content').attr('disabled', false);
      $('#title').attr('disabled', false);
      $('#tags').attr('disabled', false);
      $('#spinner').toggle(false);

      $('#paste_btn').text('Paste it!');
      $('#paste_btn').removeClass('disabled');
      $('#paste_gdrv').removeClass('disabled');
      $('#content').focus();
      $('#content').select();
    });

  });

  $('#content').bind('input propertychange', function(){
    var noc = $('#content').val().length;
    $('#noc').text(noc);

    if (noc > 1) {
      if ($('#content').parent().hasClass('has-error')) {
        $('#content').parent().removeClass('has-error');
      }
    }

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

  $('#paste_gdrv').on('click', function(event){
    event.preventDefault();

    if (!$('#content').val().length > 0) {
      $('#content').parent().addClass('has-error');
      $('#content').focus();
      return false;
    }

    if ($('input[name="destination"]').val() !== "gdrive") {
      if (auth_window != undefined) {
        auth_window.close();
      }
      var gauth_url = "/pastebin/auth/login?next=/pastebin/auth/gdrive/start"
      auth_window = window.open(gauth_url, 'gauth_frame');
    } else {
      $('#paste_gdrv').addClass('disabled');
      $('#paste_btn').click();
    }
  });

  if ($('input[name="destination"]').val() === "gdrive") {
    $('#paste_gdrv_txt').text('Pasting to Google Drive!');
    $('#paste_gdrv').data('title', 'We ken do it!!');
  };

  $('.btn').tooltip({
    placement: 'auto',
    title: $(this).data('title'),
    container: 'body'
  });

  $('#content').focus();
});

var HandleGAuthComplete = function(auth_result) {
  if (auth_result === "success") {
    $('input[name="destination"]').val("gdrive");
    $('#paste_gdrv').click();
  } else {
    $('#paste_gdrv').addClass("btn-danger");
    $('#paste_gdrv_txt').text(auth_result);
  }
};

var HandlePasteError = function(e) {
  var token_revoked = false;

  if (e.status == 401) {
    token_revoked = true
  }

  if (e.responseText.indexOf("Token has been revoked.") > 0) {
    token_revoked = true
  }

  if (token_revoked === true) { // Oops, we're unauthorized
    $('input[name="destination"]').val('datastore');
    $('#paste_gdrv_txt').text(e.statusText);
    $('#paste_gdrv').addClass("btn-danger");
    $('#paste_gdrv').tooltip('destroy');
    $('#paste_gdrv').tooltip({
      placement: 'auto',
      title: 'Click to connect again!'
    });
  }
};
