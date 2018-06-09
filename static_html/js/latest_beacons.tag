<latest_beacons>

	<style>
		/* from http://colorbrewer2.org/#type=sequential&scheme=BuGn&n=4 */
		.sig1{background-color:rgba(237,248,251, 0.65)} 
		.sig2{background-color:rgba(178,226,226, 0.65)} 
		.sig3{background-color:rgba(102,194,164, 0.65)} 
		.sig4{background-color:rgba(35,139,69, 0.65)}
		
	</style>

  <h2>{ opts.title }</h2>

	<div class="progress" id="ws_status">
    <div class="indeterminate blue"></div>
	</div>

	<div class="chip indigo accent-1" >
		{ closest.id }
	</div> 
	closest to 
	<div class="chip indigo accent-1" >
		{ closest.location }
	</div>
	with 
	<div class="chip indigo accent-1" >
		{ closest.rssi }
	</div>

  <table>
			<tr>
				<th>Beacon ID</th>
				<th>Beacon Location</th>
				<th>Beacon type</th>
				<th>Last seen</th>
				<th>Distance</th>
				<th>Add</th>
			</tr>
			<tr each={ beacon in beacons} data-is='beacon-row' item={ beacon }></tr> 
  </table>

	<br />
	<a id="pause_button" class="btn-floating btn-large waves-effect waves-light red"><i class="material-icons">pause</i></a>
	<a id="resume_button" class="btn-floating btn-large waves-effect waves-light red"><i class="material-icons">play_arrow</i></a>

  <script>
    this.beacons = opts.beacons
		this.closest = opts.closest

		var self = this;
		var ws;
		var ws_proto = "ws";
		var check_conn;
		
		var paused = false;

		if (window.location.protocol == "https:") {
			ws_proto = "wss"
		}

		var get_dist = function(distance, unit) {
			var sig = "sig1";

			if(unit == "db") {
				//console.log(distance);
				if(distance > 100) {
					sig = "sig1";
				}
				else if(distance > 90) {
					sig = "sig2";
				}
				else if(distance > 80) {
					sig = "sig3";
				}
				else {
					sig = "sig4";
				}
				//console.log(sig+" ,"+distance)
			}
			else if(unit == "meters") {
				if(distance > 15) {
					sig = "sig1";
				}
				else if(distance > 10) {
					sig = "sig2";
				}
				else if(distance > 2) {
					sig = "sig3";
				}
				else {
					sig = "sig4";
				}

			}

			return sig;
		}

		var list_filter = function(data) {
			var beacons = data.map(function(b)
				{
					b.add_label = "Add this beacon";
				if(b.beacon_type=="ibeacon") 
				{
					b.distance = Math.round(parseFloat(b.distance)*100)/100 +" meters";
					b.sig_class = get_dist((parseFloat(b.distance)*100)/100, "meters");
				}
				else if(b.beacon_type=="eddystone") 
				{
					b.distance = Math.round(parseFloat(b.distance)*100)/100 +" meters";
					b.sig_class = get_dist((parseFloat(b.distance)*100)/100, "meters");
					if(b.incoming_json.namespace == "ddddeeeeeeffff5544ff")
					{
						b.beacon_type = "HB Button";
					}
				}
				else if(b.beacon_type=="hb_button") 
				{
					b.distance = Math.round(parseFloat(b.distance)*100)/100 +" meters";
					b.sig_class = get_dist((parseFloat(b.distance)*100)/100, "meters");
					b.add_label = "Add this button";
					b.beacon_type = "HB Button"
				}
				else 
				{
					b.sig_class = get_dist(b.distance, "db");
					b.distance = "-"+b.distance+ " db";
				}
				return b;
			});

			return beacons;
		};

		var onmessage = function (evt) 
		{
			if(paused)
			{
				return;
			}

			var data = JSON.parse(evt.data);
			var data_rssi = JSON.parse(evt.data);
							
			data.sort(function(a, b) {
				//return parseInt(a.last_seen) - parseInt(b.last_seen);
				if(a.beacon_id < b.beacon_id) return -1;
				if(a.beacon_id > b.beacon_id) return 1;
				return 0;
			});
			//console.log("latest")
			//console.log(data)
			self.beacons = list_filter(data);
			
			//sort on rssi
			data_rssi = data_rssi.sort(function(a, b) {
				//return parseInt(a.last_seen) - parseInt(b.last_seen);
				if(a.incoming_json.rssi < b.incoming_json.rssi) return -1;
				if(a.incoming_json.rssi > b.incoming_json.rssi) return 1;
				return 0;
			});

			var c = data_rssi[data_rssi.length-1]
			self.closest = {
				"id": c.beacon_id, 
				"rssi": c.incoming_json.rssi,
				"location": c.beacon_location,
			}

			self.update();
		};
				
		var onopen = function()
		{
			//alert('open show');
			$("#ws_status").removeClass("hide");
		}
    
		var onclose = function()
		{ 
			// websocket is closed.
			$("#ws_status").addClass("hide");
			//alert("Connection is closed...");
		};

		beacon_ws()
		{
			if ("WebSocket" in window)
			{
				ws = new WebSocket(ws_proto+"://"+window.location.host + window.location.pathname+"api/beacons/latest/ws");
				ws.onopen = onopen;
				ws.onclose = onclose;
				ws.onmessage = onmessage;
			}
            
			else
			{
				// The browser doesn't support WebSocket
				alert("WebSocket NOT supported by your Browser!");
			}
		}

		this.on('unmount', function() {
			ws.close();
			clearInterval(check_conn);
		})

		checkConnection() {
			if(paused)
			{
				return;
			}
			if(ws.readyState == 3) {
				self.beacon_ws();
			}
		}
	
		refreshList() {
			$.getJSON( "api/latest-beacons", function( data ) {
					//console.log(data)

					data.sort(function(a, b) {
						    //return parseInt(a.last_seen) - parseInt(b.last_seen);
								if(a.beacon_id < b.beacon_id) return -1;
								if(a.beacon_id > b.beacon_id) return 1;
								return 0;
					});
					self.beacons = list_filter(data);
					
					//sort on rssi
					data.sort(function(a, b) {
							//return parseInt(a.last_seen) - parseInt(b.last_seen);
							if(a.incoming_json.rssi < b.incoming_json.rssi) return -1;
							if(a.incoming_json.rssi > b.incoming_json.rssi) return 1;
							return 0;
							});

					var c = data[data.length-1]
					self.closest = {
						"id": c.beacon_id, 
						"rssi": c.incoming_json.rssi,
						"location": c.beacon_location,
					}

					self.update()
			});
		}

		this.on('mount', function(){
			$("#ws_status").addClass("hide");
			$("#resume_button").hide();

			$("#resume_button").click(function(){
				paused = false;
				self.beacon_ws();
				$("#resume_button").hide();
				$("#pause_button").show();
			});

			$("#pause_button").click(function(){
				paused = true;
				ws.close();
				$("#pause_button").hide();
				$("#resume_button").show();
			});

			self.refreshList();
			self.beacon_ws();
			check_conn = setInterval(this.checkConnection, 1000); 		//do a thing every second
		});
		
	</script>

</latest_beacons>
				
<beacon-row>
	<td class={ beacon.sig_class } >{ beacon.beacon_id }</td>
	<td class={ beacon.sig_class } >{ beacon.beacon_location }</td>
	<td class={ beacon.sig_class }  if={beacon.beacon_type == ""} >raw</td>
	<td class={ beacon.sig_class }  if={beacon.beacon_type != ""} >{ beacon.beacon_type }</td>
	<td class={ beacon.sig_class } >{ moment(beacon.last_seen*1000).fromNow() }</td> 
	<td class={ beacon.sig_class }  if={beacon.distance != ""} >{ beacon.distance }</td>
	<td class={ beacon.sig_class } ><a href="#add-beacon/{ beacon.beacon_id }">{ beacon.add_label }</a></td>
</beacon-row>
