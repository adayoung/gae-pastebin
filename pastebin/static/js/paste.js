function scale_iframe(){
  $('iframe#content').css('overflow', 'hidden');

  // Because Firefox still shows scrollbars
  $('iframe#content').attr('scrolling', 'no');

  // the two lines below need to be jQuery'd :o
  var i_r_iframe = document.getElementById('content');
  try { // lol
    if (i_r_iframe.contentDocument.body) {
      i_r_iframe.height = i_r_iframe.contentDocument.body.scrollHeight + 30 + 'px';
    }
  } catch (e) {
    console.log(e);
    return false;
  }
}

$('iframe#content').on('load', function(){
  scale_iframe();
});

$(document).ready(function(){
  $.each($('.btn'), function(){
    $(this).tooltip();
  });

  $('#deletebtn').on('click', function(event){
    event.preventDefault();

    $('#deletebtn').addClass('disabled');
    $.post(location.href + "/delete", {
      delete: "yes",
      "gorilla.csrf.Token": $('input[name="gorilla.csrf.Token"]').val()
    }).done(function(data) {
      alert("Okay! This paste is no longer available.");
      location.replace(data);
    }).fail(function(e){
      alert("Oops, we couldn't delete this paste :( The following was encountered:\n\n" + e.status + " - " + e.statusText);
      location.reload();
    }).always(function(e){
      $('#deletebtn').removeClass('disabled');
    });
  });
});