<%@ page contentType="text/html;charset=UTF-8" %>
	<%@ include file="/common/taglibs.jsp" %>
		<style>
			.group-title {
				margin: 20px 0 10px 25px
			}
			
			.slide-content {
				padding: 0px !important
			}
			
			.power-system {
				border-bottom: #E9E9E9 solid 1px
			}
			
			.power-system li {
				margin-left: 50px;
				margin-bottom: 15px
			}
			
			.power-system li lable {
				color: #9E9E9E;
				width: 150px;
				font-size: 14px;
				display: inline-block
			}
			
			#viewPciTaskEnb {
				font-size: 14px;
			}
			
			.sfp-wait .el-icon-status-SFP:before,
			.lan-wait .el-icon-status-LAN:before {
				color: #CFCFCF
			}
			
			.el-icon-status-powerON:before,
			.sfp-success .el-icon-status-SFP:before,
			.wendu-low .el-icon-status-temperature:before,
			.el-icon-status-high:before,
			.lan-success .el-icon-status-LAN:before,
			.status-normal .el-icon-status-run:before,
			.el-icon-status-charging:before{
				color: #67D972
			}
			.status-normal .el-icon-status-run:before{
				position: relative;
				left: -2px
			}
			.el-icon-status-powerOFF:before,
			.sfp-error .el-icon-status-SFP:before,
			.wendu-height .el-icon-status-temperature:before,
			.el-icon-status-low:before,
			.lan-error .el-icon-status-LAN:before,
			.el-icon-status-fault:before {
				color: #E88282
			}
			.el-icon-status-run:before{
				color:#F2B354
			}
			.el-icon-status-discharging:before{
				color:#4D84FF;
				position: relative;
				left:-3px
			}
			.el-icon-status-standby:before{
				color:#363B4E
			}
			.dac {
				display: contents;
			}
			
			.ml3 {
				margin-left: 3px
			}
			
			.f14 {
				font-size: 14px
			}
			
			.spanTwo {
				color: #9E9E9E;
				font-size: 12px;
				margin-left: 15px
			}
			.temperCls{
				display:inline-block;
			}
		</style>
		<div id='viewPciTaskEnb'>

			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("DianYuanXiTongCanShu") %></span>
			</div>
			<ul class="power-system">
				<li>
					<lable><%=rb.getString("ChangShang") %></lable><span class="f14">{{powerSystem.manufacturer}}</span>
				</li>
				<li>
					<lable><%=rb.getString("ChangShang") %>OUI</lable> <span class="f14">{{powerSystem.manufacturer_oui}}</span>
				</li>
				<li>
					<lable><%=rb.getString("DianYuanBianMa") %></lable> <span class="f14">{{powerSystem.serial_number}}</span>
				</li>
				<li>
					<lable><%=rb.getString("YingJianBanBen") %></lable> <span class="f14">{{powerSystem.hardware_version}}</span>
				</li>
				<li>
					<lable><%=rb.getString("SoftwareVersion") %></lable> <span class="f14">{{powerSystem.software_version}}</span>
				</li>
				<li>
					<lable>UpTime</lable> <span class="f14" v-if='powerSystem.uptime === "--"'>--</span>
					 <span class="f14" v-else>{{powerSystem.uptime}}</span>
				</li>
				<li>
					<lable><%=rb.getString("ChanPinXingHao") %></lable> <span class="f14">{{powerSystem.product_class}}</span>
				</li>

			</ul>
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("DianYuanYunXingCanShu") %></span>
			</div>
			<ul class="power-system">
				<li>
					<lable><%=rb.getString("ShiDianZhuangTai") %></lable> <span class="f14">
					<div v-if="poweroperation.ac_power == 'ON'" class="dac" >
						<span class="el-icon el-icon-status-powerON"><span class="f14">{{poweroperation.ac_power}}</span></span>
					</div>
					<div v-if="poweroperation.ac_power == 'OFF'" class="dac">
						<span class="el-icon el-icon-status-powerOFF"> <span class="f14">{{poweroperation.ac_power}}</span> </span>
					</div>
					<div v-if="poweroperation.ac_power == '--'" class="dac">
						<span>--</span>
					</div>
		</li>
		<li>
			<lable><%=rb.getString("ACVoltage") %></lable> <span class="f14" v-if="poweroperation.ac_voltage==='--'">--</span> 
			<span class="f14" v-else>{{poweroperation.ac_voltage}}V</span>
		</li>
		<li>
			<lable><%=rb.getString("DCVoltage") %></lable> <span class="f14" v-if="poweroperation.dc_voltage==='--'">--</span>
			<span class="f14" v-else>{{poweroperation.dc_voltage}}V</span>
		</li>
		<li>
			<lable><%=rb.getString("DCCurrent") %></lable><span class="f14" v-if="poweroperation.dc_current==='--'"> --</span>
			<span class="f14" v-else>{{poweroperation.dc_current}}A</span>
		</li>
		<li>
			<lable><%=rb.getString("BoardTemperature") %></lable>
			<div v-if='poweroperation.board_temp <= 0' class="wendu-height dac" >
				<span class="el-icon el-icon-status-temperature"><span class="f14">{{poweroperation.board_temp}}℃</span></span>
			</div>
			<div v-if='poweroperation.board_temp > 0 && poweroperation.board_temp <= 10' class="wendu-middle dac">
				<span class="el-icon el-icon-status-temperature"><span class="f14">{{poweroperation.board_temp}}℃</span></span>
			</div>
			<div v-if='poweroperation.board_temp > 10 && poweroperation.board_temp <= 40' class="wendu-low dac">
				<span class="el-icon el-icon-status-temperature"><span class="f14">{{poweroperation.board_temp}}℃</span></span>
			</div>
			<div v-if='poweroperation.board_temp > 40 && poweroperation.board_temp < 60' class="wendu-middle dac">
				<span class="el-icon el-icon-status-temperature"><span class="f14">{{poweroperation.board_temp}}℃</span></span>
			</div>
			<div v-if='poweroperation.board_temp >=60' class="wendu-height dac">
				<span class="el-icon el-icon-status-temperature"><span class="f14">{{poweroperation.board_temp}}℃</span></span>
			</div>
			<div v-if='poweroperation.board_temp ==="--"' class="wendu-height dac">
				<span>--</span>
			</div>
		</li>
		<li>
			<lable><%=rb.getString("SFPState") %></lable>
			<div v-if='poweroperation.sfp_state === "DISCONNECT"' class="sfp-wait dac">
				<span class="el-icon  el-icon-status-SFP"></span>
				<span class="f14 ml3">{{poweroperation.sfp_state}}</span>
			</div>
			<div v-if='poweroperation.sfp_state === "PLUGIN"' class="sfp-success dac">
				<span class="el-icon  el-icon-status-SFP"></span>
					<span class="f14 ml3">{{poweroperation.sfp_state}}</span>
			</div>
			<div v-if='poweroperation.sfp_state === "FAULT"' class="sfp-error dac">
				<span class="el-icon  el-icon-status-SFP"></span>
					<span class="f14">{{poweroperation.sfp_state}}</span>
			</div>
			<div v-if='poweroperation.sfp_state === "--"' class="sfp-error dac">
				<span>--</span>
			</div>
		</li>
		<li>
			<lable><%=rb.getString("Port0State") %></lable>
			<div class=" lan-success dac" v-if='poweroperation.lan_0_state == "UP"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_0_state}}</span>
			</div>
			<div class=" lan-error dac" v-if='poweroperation.lan_0_state == "FAULT"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_0_state}}</span>
			</div>
			<div class=" lan-wait dac" v-if='poweroperation.lan_0_state == "DOWN"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_0_state}}</span>
			</div>
			<div  class=" lan-wait dac" v-if='poweroperation.lan_0_state == "--"'>
				<span>--</span>
			</div>
		</li>
		<li>
			<lable><%=rb.getString("Port1State") %></lable>
			<div class=" lan-success dac" v-if='poweroperation.lan_1_state == "UP"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_1_state}}</span>
			</div>
			<div class=" lan-error dac" v-if='poweroperation.lan_1_state == "FAULT"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_1_state}}</span>
			</div>
			<div class=" lan-wait dac" v-if='poweroperation.lan_1_state == "DOWN"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_1_state}}</span>
			</div>
			<div class=" lan-wait dac" v-if='poweroperation.lan_1_state == "--"'>
				<span>--</span>
			</div>
		</li>
		<li>
			<lable><%=rb.getString("Port2State") %></lable>
			<div class=" lan-success dac" v-if='poweroperation.lan_2_state == "UP"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_2_state}}</span>
			</div>
			<div class=" lan-error dac" v-if='poweroperation.lan_2_state == "FAULT"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_2_state}}</span>
			</div>
			<div class=" lan-wait dac" v-if='poweroperation.lan_2_state == "DOWN"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_2_state}}</span>
			</div>
			<div class=" lan-wait dac" v-if='poweroperation.lan_2_state == "--"'>
				<span>--</span>
			</div>
		</li>
		<li>
			<lable><%=rb.getString("Port3State") %></lable>
			<div class=" lan-success dac" v-if='poweroperation.lan_3_state == "UP"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_3_state}}</span>
			</div>
			<div class=" lan-error dac" v-if='poweroperation.lan_3_state == "FAULT"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_3_state}}</span>
			</div>
			<div class=" lan-wait dac" v-if='poweroperation.lan_3_state == "DOWN"'>
				<span class="el-icon el-icon-status-LAN"></span>
				<span class="f14">{{poweroperation.lan_3_state}}</span>
			</div>
			<div class=" lan-wait dac" v-if='poweroperation.lan_3_state == "--"'>
				<span>--</span>
			</div>
		</li>
		<li>
			<lable><%=rb.getString("AverageSOS") %></lable>
			<div v-if='poweroperation.ave_soc >=0 && poweroperation.ave_soc <= 40' class="dac">
				<span class="el-icon el-icon-status-low">
								<span class="f14">{{poweroperation.ave_soc}}%</span>
				</span>
			</div>
			<div v-if='poweroperation.ave_soc >40 && poweroperation.ave_soc <= 70' class="dac">
				<span class="el-icon el-icon-status-run">
							<span class="f14">{{poweroperation.ave_soc}}%</span>
				</span>
			</div>
			<div v-if='poweroperation.ave_soc >70 && poweroperation.ave_soc <= 100' class="dac">
				<span class="el-icon el-icon-status-high">
							<span class="f14">{{poweroperation.ave_soc}}%</span>
				</span>
			</div>
			<div v-if='poweroperation.ave_soc==="--"' class="dac">
				<span>--</span>
			</div>
		</li>
		<li>
			<lable><%=rb.getString("DianChiGeShu") %></lable>
			<span v-if='poweroperation.pack_counts === "--"'>
				--
			</span>
			<span v-else>
				{{poweroperation.pack_counts}}
			</span>
		</li>
		</ul>
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("DianChiYunXingCanShu") %></span>
		</div>
		<el-ctable style='margin-left:50px;' ref="view_enb" :rownumber="true" id="viewEnb_table"  height="300px" :pagination="false" :data='listdata'>
			<el-table-column label='<%=rb.getString("SOC") %>' prop="soc" width='200'>
				<template slot-scope="scope">
					<div v-for="(item,index) in scope.row.soc.split(',')" style='display:inline-block'>
						<div v-if='item >=0 && item <= 40' style='display:inline-block'>
							<span class="el-icon el-icon-status-low">
								<span class="f12">{{item}}%</span>
							</span>
						</div>
						<div v-if='item >40 && item <= 70' style='display:inline-block'>
							<span class="el-icon el-icon-status-run">
								<span class="f12">{{item}}%</span>
							</span>
						</div>
						<div v-if='item >70 && item <= 100' style='display:inline-block'>
							<span class="el-icon el-icon-status-high">
								<span class="f12">{{item}}%</span>
							</span>
						</div>
						<span v-if="index < scope.row.soc.split(',').length-1">,</span>
					</div>
					</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("DianYa") %>' prop='voltage' width="150"></el-table-column>
			<el-table-column label='<%=rb.getString("WenDu") %>' prop='temperature' width="250">
				<template slot-scope="scope">
						<div v-for="(item,index) in scope.row.temperature.split(',')" style='display:inline-block'>
							<div v-if='item <= 0' class="wendu-height temperCls">
								<span class="el-icon el-icon-status-temperature"><span class="f12">{{item}}</span></span>
							</div>
							<div v-if='item > 0 && scope.row.temperature <= 10' class="wendu-middle temperCls">
								<span class="el-icon el-icon-status-temperature"><span class="f12">{{item}}</span></span>
							</div>
							<div v-if='item > 10 && item <= 40' class="wendu-low temperCls">
								<span class="el-icon el-icon-status-temperature"><span class="f12">{{item}}</span></span>
							</div>
							<div v-if='item > 40 && item < 60' class="wendu-middle temperCls">
								<span class="el-icon el-icon-status-temperature"><span class="f12">{{item}}</span></span>
							</div>
							<div v-if='item >=60' class="wendu-height temperCls">
								<span class="el-icon el-icon-status-temperature"><span class="f12">{{item}}</span></span>
							</div>
							<span v-if="index < scope.row.temperature.split(',').length-1">,</span>
						</div>
				
				
						<%-- <div v-if='scope.row.temperature <= 0' class="wendu-height">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.temperature}}</span></span>
						</div>
						<div v-if='scope.row.temperature > 0 && scope.row.temperature <= 10' class="wendu-middle">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.temperature}}</span></span>
						</div>
						<div v-if='scope.row.temperature > 10 && scope.row.temperature <= 40' class="wendu-low">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.temperature}}</span></span>
						</div>
						<div v-if='scope.row.temperature > 40 && scope.row.temperature < 60' class="wendu-middle">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.temperature}}</span></span>
						</div>
						<div v-if='scope.row.temperature >=60' class="wendu-height">
							<span class="el-icon el-icon-status-temperature"><span class="f12">{{scope.row.temperature}}</span></span>
						</div> --%>
					</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("DianLiu") %>' prop="current">
				<template slot-scope="scope">
					{{scope.row.current}}(MA)
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai") %>' prop="status">
				<template slot-scope="scope">
					<div class="status-normal" v-if='scope.row.status === "NORMAL"'>
						<span class="el-icon el-icon-status-run"><span class="f12">{{scope.row.status}}</span></span>
					</div>
					<div  v-else>
						<span class="el-icon el-icon-status-fault"><span class="f12">{{scope.row.status}}</span></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("RecycleCount") %> ' prop="recyle_count"></el-table-column>
			
			<el-table-column label='<%=rb.getString("Charging") %>' prop="charging">
				<template slot-scope="scope">
					<div v-if='scope.row.charging === "CHARGING"'>
						<span class="el-icon el-icon-status-charging"><span class="f12">{{scope.row.charging}}</span></span>
					</div>
					<div v-if='scope.row.charging === "DISCHARGING"'>
						<span class="el-icon el-icon-status-discharging"><span class="f12">{{scope.row.charging}}</span></span>
					</div>
					<div v-if='scope.row.charging === "STANDBY"' class="standby">
						<span class="el-icon el-icon-status-high"><span class="f12">{{scope.row.charging}}</span></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("SheBeiXingHao") %>' prop="model"></el-table-column>
			<el-table-column label='<%=rb.getString("SoftwareVersion") %>' prop="software_version" min-width="130"></el-table-column>
			<el-table-column label='<%=rb.getString("DianYuanBianMa") %> ' prop="serial_number"></el-table-column>
		</el-ctable>
		</div>

		<script>
			var addEnb = new Vue({
				el: '#viewPciTaskEnb',
				data() {
					return {
						powerSystem: {},
						poweroperation: {},
						url: '',
						listdata:[]

					}
				},
				methods: {
					/**
					* 获取详情信息
					**/
					init(data) { 
						var vm = this,
							url= '${ctx}/ups/queryInformations.action',
							obj= {
								ups_code:data.ups_code
							};
							
						axios.post(url, stringify(obj)).then(function(res){
							if(res.status=== 200){
								var powerSystem = res.data.power_system_param,
									poweroperation = res.data.power_run_param,
									listdata = res.data.battery_run_param;
									
								vm.powerSystem = powerSystem;
								vm.poweroperation = poweroperation;
								vm.listdata = listdata
							}else{
								vm.poweroperation ={}
								vm.powerSystem ={}
								vm.listdata=[]
							}
						})
					}

				},
				mounted() {
					eventBus.$off('open-dialog').$on('open-dialog', this.init);
					eventBus.$off('cancel-slide').$on('cancel-slide', this.cancel);
				}
			})
		</script>