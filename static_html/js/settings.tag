<settings>
  <h2>{ opts.title }</h2>

	<div class="row">
    <form class="col s12" name="settings-form" id="settings-form">
      <div class="row">
        <div class="input-field col s12">
          <input value="{opts.location_confidence}" id="location_confidence" type="number" min="1">
          <label for="location_confidence" class="active">Location Confidence<i class="material-icons tooltipped" data-position="right" data-tooltip="How many times in a row a beacon should be seen in a location before it's considered located there. The higher this value is, the more accuracy is improved, but the longer it will need to take effect">help</i></label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input value="{opts.last_seen_threshold}" id="last_seen_threshold" type="number" min="1">
          <label for="last_seen_threshold" class="active">Last Seen Threshold<i class="material-icons tooltipped" data-position="right" data-tooltip="How many seconds until a beacon is considered gone">help</i></label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input value="{opts.last_reading_threshold}" id="last_reading_threshold" type="number" min="1">
          <label for="last_reading_threshold" class="active">Last Reading Threshold<i class="material-icons tooltipped" data-position="right" data-tooltip="Advanced option. How many seconds until a beacon reading is removed from calculations">help</i></label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input checked="{opts.ha_send_changes_only}" id="ha_send_changes_only" type="checkbox">
          <label for="ha_send_changes_only" class="active">Only Send Changes in Location to Home Assistant<i class="material-icons tooltipped" data-position="right" data-tooltip="Only send changes to Home Assistant when the location of a beacon changes">help</i></label>
        </div>
      </div>
      <div class="row">
				<button class="btn waves-effect waves-light" type="submit" name="action">Submit
		  		<i class="material-icons right">send</i>
		   </button> 
      </div>
    </form>
  </div>

	<script>
		this.on('mount', function() {

			$('.tooltipped').tooltip({delay: 20});

			$.ajax(
			{
				url: "api/settings",
				type: "GET",
				dataType: 'json',
			})
			.done(function(data) {
				if(data.last_reading_threshold) {
					$("#last_reading_threshold").val(data.last_reading_threshold);
				}
				if(data.location_confidence) {
					$("#location_confidence").val(data.location_confidence);
				}
				if(data.last_seen_threshold) {
					$("#last_seen_threshold").val(data.last_seen_threshold);
				}
				if(data.ha_send_changes_only) {
					$("#ha_send_changes_only").prop("checked", data.ha_send_changes_only);
				}
			});

			$("#settings-form").submit(function(event){
				event.preventDefault();
				var form_data = {}
				form_data["location_confidence"] = parseInt($("#location_confidence").val());
				form_data["last_seen_threshold"] = parseInt($("#last_seen_threshold").val());
				form_data["last_reading_threshold"] = parseInt($("#last_reading_threshold").val());
				form_data["ha_send_changes_only"] = $("#ha_send_changes_only").prop("checked");
				//console.log(form_data);
				$.ajax(
				{
					url: "api/settings",
					type: "POST",
					contentType: 'application/json; charset=UTF-8',
					data: JSON.stringify(form_data),
				})
				.done(function(data) {
					window.location.hash = '#home';
				});
			});
		})
	</script>

</settings>

