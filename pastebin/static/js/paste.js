var scale_iframe = function(tehframe){
  $(tehframe).css('overflow', 'hidden');

  // the lines below need to be jQuery'd :o
  try {
    if (tehframe.contentDocument.body) {
      $(tehframe).css('height', tehframe.contentDocument.body.scrollHeight + 30);
    }
  } catch (e) {}
};

$('iframe').on('load', function(){
  scale_iframe(this);
});

var meep = function(l, f){
  $(l).append("<span>.Meep! I couldn't get the content -flails- ("+f.responseText+")</span>");
  console.log(f);
}

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
      alert("BAM!@ Okay! This paste is no longer available.");
      location.replace(data);
    }).fail(function(e){
      alert("Oops, we couldn't delete this paste :( The following was encountered:\n\n" + e.status + " - " + e.statusText);
      location.reload();
    }).always(function(e){
      $('#deletebtn').removeClass('disabled');
    });
  });

  if ($('input[name=format]').val() === "html") {

    var content = document.createElement('iframe');
    content.sandbox="allow-same-origin";
    $('article').append(content);
    $(content).on('load', function(){
      scale_iframe(content);
    });

    if ($('#driveHosted').length > 0) {
      var loader = document.createElement('p');
      $(loader).html('<span>Loading content.. Please wait.</span> <img alt="pretty spinner" src="/pastebin/static/img/spinner.gif">');
      $('article').append(loader);
      $.get(location.href+'/content/link', function(src){
        $(loader).append("<span>.</span>");
        $.get(src, function(data){
          var blob = new Blob([data], {type: 'text/html'});
          var url = URL.createObjectURL(blob);
          content.src = url;
          $(loader).append("<span>.done!</span>");
          $(loader).slideUp();
        }).fail(function(f){
          meep(loader, f);
        });
      }).fail(function(f){
          meep(loader, f);
      });
    } else {
      var paste_id = location.href.split('/pastebin/')[1];
      content.src="/pastebin/"+paste_id+"/content";
    }
  } else {
    $("pre").html(function (index, html) {
      return html.replace(/^(.*)$/mg, "<span class=\"line\">$1</span>")
    });
  }
});
