<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	#cpeUpgradeAddTask{
		overflow-x: hidden;
	}
	#cpeUpgradeAddTask .modeItem .el-radio{
		display:block;
		margin-left:0px;
		margin-bottom:20px;
	} 
	#cpeUpgradeAddTask .modeItem{
		margin-top:20px;
	}
	#cpeUpgradeAddTask .tableDiv{
		height:15px;
		width:auto;
	}
	.tableSelectCls{
		cursor: pointer;
		text-align: center;
	}
	.tableSelectCls .el-icon::before{
		color: #1EBC1E;
	}
	#cpeUpgradeAddTask .el-select .el-input.is-disabled .el-input__inner{
		min-height:26px;
		max-height:26px;
	}
	.deviceItem{
		display:inline-block;
		margin-right:60px;
	}
	.alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	.titleStyML{
		margin-left: 20px;
	}
	.filterFile{
		margin-left: 15px;
	}
	.filterFile .el-checkbox__label {
		font-size:12px;
		font-weight:normal;
		margin-left:0;
		color: #4D84FF;
	}
	.deviceTableBox{
		margin:10px 0px 0px 45px;
		display: flex;
	}
	.deviceSpecifiedBox{
		width: 200px;
		height: 370px;
		border:1px solid #E9E9E9;
		border-right: none;
	}
	.deviceSpecifiedTitle{
		height: 36px;
		width: 200px;
		font-size: 12px;
		line-height: 36px;
		text-align: center;
		background: #F6F7FB;
		box-sizing: border-box;
		border-bottom:1px solid #E9E9E9;
	}
	.specifiedTypeBox{
		flex: 1;
		padding-top: 30px;
		padding-left: 40px;
	}
	.specifiedTypeBox .el-radio__label{
		font-size: 12px !important;
	}
	.specifiedTypeBox .el-radio+.el-radio{
		margin-left: 0px;
		display: block;
	}
	.cpeDeviceTableBox{
		width: 100% !important;
		font-size: 12px !important;
	}
	.cpeDeviceTableBox .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	.cpeDeviceTableBox .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	#cpeUpgradeAddTask .cpeDeviceTableBox .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	.tableTitles{
		margin-left: 45px;
		font-size: 14px;
	}
	#cpeUpgradeAddTask .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	#cpeUpgradeAddTask .el-radio__label{
		font-size:12px;
	}
	#cpeUpgradeAddTask .editButton{
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
		position: absolute;
		right: 140px;
		top: 15px;
	}
	#cpeUpgradeAddTask .editButton i{
		font-size:14px !important;
	}
	#cpeUpgradeAddTask .editButton span{
		font-size:12px;
	}
	.dialogStyle .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	#cpeUpgradeAddTask .el-form-item__label{
		line-height: 26px;
	}
	#cpeUpgradeAddTask .el-form-item__content{
		margin-left: unset!important;
		width: 100%;
	}
	#cpeUpgradeAddTask .el-ctable-toolbar{
		padding: 10px 0 !important;
	}
</style>

<!-- 新建cpe升级任务 -->
<div id="cpeUpgradeAddTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" style='padding-top:20px;' label-width="165px" label-position="left"  :hide-required-asterisk=true>
		<div class="group-title not-extend titleStyML" >
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname' style='margin-left:45px;margin-top:20px;'>
			<el-input maxlength=100 :disabled='viewFlag' v-model="ruleForm.taskname" size="mini" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<el-form-item label='Upgrade Type' prop='upgradeMode' style='margin-left:45px;margin-top:20px;'>
			<el-radio-group v-model="ruleForm.upgradeMode" @change="modeChange" :disabled='viewFlag'>
				<el-radio border label="image">Image Upgrade</el-radio>
				<el-radio border label="module">Module Upgrade</el-radio>
			</el-radio-group>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div>
			<div class="group-title not-extend titleStyML" >
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
			</div>

			<el-form-item v-if="ruleForm.upgradeMode == 'module'" label='<%=rb.getString("MoKuaiMingCheng")%>' style="margin:20px 0px 20px 45px;">
				<el-select v-model="ruleForm.moduleName" size="mini" @change="moduleChange" :disabled='viewFlag'>
					<el-option label="All" value=""></el-option>
					<el-option v-for="item in moduleNameList" :label="item.module_name" :value="item.module_name"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="<%=rb.getString("ZaiXianZhuangTai")%>" prop="connection_status" style="margin:20px 0 20px 45px;" v-show='!viewFlag && false'>
				<el-select v-model="ruleForm.connection_status" @change="connectionStatusChange">
					<el-option v-for="item in onlineOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='<%=rb.getString("SheBeiZhiDing")%>' style="margin:20px 0px 0px 45px;">
				<el-radio-group  v-model="ruleForm.deviceAssign" :disabled="viewFlag" @change="deviceAssignChange">
					<el-radio border  label="1"><%=rb.getString("QuanBu")%></el-radio>
					<el-radio border  label="0"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>

			<div class="deviceTableBox">
				<div class="cpeDeviceTableBox">
					<el-pairgrid 
						:id="'select_device_list'" 
						v-if="showPairGrid && ruleForm.deviceAssign == '0'" 
						style="margin-right:45px;" query-name="mac_address" 
						:rownumber="true" 
						ref="cpeDevicePairgrid"  
						@selection-change='selectChange' 
						:right-url="rightUrl" :left-url="leftUrl" 
						:height="height" row-key="small_cell_code" 
						:query-params="queryParams" :title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("CPEMacAddress")%>'}"
					>
						<template slot="left">
							<el-table-column type="selection" width="45"></el-table-column>
							<el-table-column prop="connection_status" width="50">
								<template slot-scope="scope">
									<div :class="{
										'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
										'':scope.row.have_connected==2,
										'conn_exc':scope.row.connection_status=='Exception',
										'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
								</template>
							</el-table-column>
							<el-table-column prop="small_cell_code" v-if="false"></el-table-column>
							<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="180"></el-table-column>
							<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="160" ></el-table-column>
							<el-table-column prop="cpe_device_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="220" ></el-table-column>
							<el-table-column prop='ipaddress' label='IP' width="180"></el-table-column>
							<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" min-width="160" ></el-table-column>
							<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  min-width="160"></el-table-column>
							<el-table-column prop="model_name" sortable show-overflow-tooltip label="<%=rb.getString("ChanPinXingHao")%>" min-width="160" ></el-table-column>
							<el-table-column prop="cell_name" label="<%=rb.getString("HostName")%>" min-width="160" sortable></el-table-column>
							<el-table-column prop="cell_identity" label="ECI" min-width="80" ></el-table-column>
							<el-table-column prop="pci" label="PCI" min-width="60" ></el-table-column>
							<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="160" ></el-table-column>
							<el-table-column prop="moduleName" show-overflow-tooltip label="<%=rb.getString("MoKuaiMingCheng")%>" min-width="110" ></el-table-column>
							<el-table-column prop="moduleVersion" show-overflow-tooltip label="<%=rb.getString("MoKuaiBanNen")%>" min-width="160" ></el-table-column>
						</template>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>

							<div style="display: flex;align-items: center;">
								<el-query type="normal" @query="query" @advance-query="advanceQuery" @reset='resetQuery' placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("HostName")%> / <%=rb.getString("IMSI")%> / PCI / IP"
									:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<template v-if="false" slot="form">
										<el-form :model='queryForm' ref="queryForm" label-position="top" class="flex-form">
											<el-form-item label='<%=rb.getString("CPEBianMa")%>' prop='serial_number'>
												<el-input v-model='queryForm.serial_number' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='<%=rb.getString("IMSI")%>' prop='imsi'>
												<el-input v-model='queryForm.imsi' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='<%=rb.getString("CPEName")%>' prop='host_name'>
												<el-input v-model='queryForm.host_name' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
												<el-select v-model="queryForm.group_id" size="mini" multiple collapse-tags>
													<el-option v-for="item in deviceGroups" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item label='<%=rb.getString("BanBen")%>' prop='software_version'>
												<el-select v-model="queryForm.software_version" size="mini" multiple collapse-tags>
													<el-option v-for="item in versions" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item label='<%=rb.getString("ChanPinXingHao")%>' prop='model_name'>
												<el-select v-model="queryForm.model_name" size="mini" filterable multiple collapse-tags>
													<el-option v-for="item in modelName" :label="item.model_name" :value="item.model_name">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item label='<%=rb.getString("HostName")%>' prop='cell_name'>
												<el-input v-model='queryForm.cell_name' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='PCI' prop='pci'>
												<el-input v-model='queryForm.pci' size="mini"></el-input>
											</el-form-item>
										</el-form>
									</template>
								</el-query>
								
								<el-popfilter style="margin: 0 5px;"
									label="<%=rb.getString("SheBeiZu")%>"
									v-model="queryForm.group_id"
									:list="deviceGroups.filter(item=>item.value !== '').map(item => ({label: item.text, value: item.value}))"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;"
									v-show='!viewFlag'
									type="single"
									label="<%=rb.getString("ZaiXianZhuangTai")%>"
									v-model="queryForm.connection_status"
									:list="onlineOptions"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;"
									label="<%=rb.getString("BanBen")%>"
									v-model="queryForm.software_version"
									:list="versions.filter(item=>item.value !== '').map(item => ({label: item.text, value: item.value}))"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;"
									label="<%=rb.getString("ChanPinXingHao")%>"
									v-model="queryForm.model_name"
									:list="modelName.map(item => ({label: item.model_name, value: item.model_name}))"
									@check-change="advanceQuery">
								</el-popfilter>

								<div class="pop-filter-clear" 
									@click="resetQuery">
									<%=rb.getString("QingKongShaiXuan")%>
								</div>
							</div>

							<div class="editButton" size="mini" @click="addBatchSn">
								<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
								<span><%=rb.getString("PiLiangShuRu")%></span>
							</div>
						</template>
						<template slot='right'>
							<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>"></el-table-column>
							<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>"></el-table-column>
							<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" min-width="200" ></el-table-column>
						</template>
					</el-pairgrid>
					<el-ctable 
						:id="'all_device_list'" 
						v-if="ruleForm.deviceAssign == '1'" 
						:query-params="queryParams" 
						style="border:1px solid #E9E9E9;margin-right:45px;" 
						ref="cpeDevicePairgrid"
						:url="leftUrl" 
						:height="height" 
						pagination="true" 
					>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>

							<div style="display: flex;align-items: center;">
								<el-query type="normal" @query="query" @advance-query="advanceQuery" @reset='resetQuery' placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("HostName")%> / <%=rb.getString("IMSI")%> / PCI / IP"
									:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<template v-if="false" slot="form">
										<el-form :model='queryForm' ref="queryForm" label-position="top" class="flex-form">
											<el-form-item label='<%=rb.getString("CPEBianMa")%>' prop='serial_number'>
												<el-input v-model='queryForm.serial_number' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='<%=rb.getString("IMSI")%>' prop='imsi'>
												<el-input v-model='queryForm.imsi' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='<%=rb.getString("CPEName")%>' prop='host_name'>
												<el-input v-model='queryForm.host_name' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
												<el-select v-model="queryForm.group_id" size="mini" multiple collapse-tags>
													<el-option v-for="item in deviceGroups" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item label='<%=rb.getString("BanBen")%>' prop='software_version'>
												<el-select v-model="queryForm.software_version" size="mini" filterable multiple collapse-tags>
													<el-option v-for="item in versions" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item label='<%=rb.getString("ChanPinXingHao")%>' prop='model_name'>
												<el-select v-model="queryForm.model_name" size="mini" filterable multiple collapse-tags>
													<el-option v-for="item in modelName" :label="item.model_name" :value="item.model_name">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item label='<%=rb.getString("HostName")%>' prop='cell_name'>
												<el-input v-model='queryForm.cell_name' size="mini"></el-input>
											</el-form-item>
											<el-form-item label='PCI' prop='pci'>
												<el-input v-model='queryForm.pci' size="mini"></el-input>
											</el-form-item>
										</el-form>
									</template>
								</el-query>

								<el-popfilter style="margin: 0 5px;"
									label="<%=rb.getString("SheBeiZu")%>"
									v-model="queryForm.group_id"
									:list="deviceGroups.filter(item=>item.value !== '').map(item => ({label: item.text, value: item.value}))"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;"
									v-show='!viewFlag'
									type="single"
									label="<%=rb.getString("ZaiXianZhuangTai")%>"
									v-model="queryForm.connection_status"
									:list="onlineOptions"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;"
									label="<%=rb.getString("BanBen")%>"
									v-model="queryForm.software_version"
									:list="versions.filter(item=>item.value !== '').map(item => ({label: item.text, value: item.value}))"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;"
									label="<%=rb.getString("ChanPinXingHao")%>"
									v-model="queryForm.model_name"
									:list="modelName.map(item => ({label: item.model_name, value: item.model_name}))"
									@check-change="advanceQuery">
								</el-popfilter>

								<div class="pop-filter-clear" 
									@click="resetQuery">
									<%=rb.getString("QingKongShaiXuan")%>
								</div>
							</div>
						</template>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop="small_cell_code" v-if="false"></el-table-column>
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="180"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="160" ></el-table-column>
						<el-table-column prop="cpe_device_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="220" ></el-table-column>
						<el-table-column prop='ipaddress' label='IP' width="180"></el-table-column>
						<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" min-width="160" ></el-table-column>
						<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  min-width="160"></el-table-column>
						<el-table-column prop="moduleName" show-overflow-tooltip label="<%=rb.getString("MoKuaiMingCheng")%>"  min-width="160"></el-table-column>
						<el-table-column prop="model_name" sortable show-overflow-tooltip label="<%=rb.getString("ChanPinXingHao")%>" min-width="160" ></el-table-column>
						<el-table-column prop="cell_name" label="<%=rb.getString("HostName")%>" min-width="160" sortable></el-table-column>
						<el-table-column prop="cell_identity" label="ECI" min-width="80" ></el-table-column>
						<el-table-column prop="pci" label="PCI" min-width="60" ></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
						<el-table-column prop="moduleVersion" show-overflow-tooltip label="<%=rb.getString("MoKuaiBanNen")%>" min-width="160" ></el-table-column>
					</el-ctable>
					<div v-if="!showPairGrid && ruleForm.deviceAssign == '0'" >
						<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="fileParams">
							<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="200"></el-table-column>
							<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>"></el-table-column>
							<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" min-width="200" ></el-table-column>
							<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>" min-width="100"></el-table-column>
						</el-ctable>
					</div>
				</div>
			</div>
			
			<el-form-item prop='cellCodes' style='margin-left:45px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
		</div>
		<div class="tableTitles">
			<span ><%=rb.getString("WenJianLieBiao")%></span>
			<el-checkbox class="filterFile" 
				v-model="noFilterFile" 
				:disabled='viewFlag' 
				@change="filterFileChange" 
				label="<%=rb.getString("ZiDongGenJuMoKuaiXingHaoGuoLv")%>"></el-checkbox>

			<el-checkbox class="filterFile" v-if="ruleForm.upgradeMode == 'image'"
				v-model="ruleForm.needModuleUpgrade" 
				true-label="true"
				false-label="false"
				:disabled='viewFlag'
				@change="filterModuleFileChange"
				label="Auto Upgrade related Module version"></el-checkbox>
		</div>
		<div class='fileContent' style='margin:10px 45px 0px 45px;border:1px solid #E9E9E9;'>
			<el-ctable :id="'select_file_list'" row-key="id" 
				:readonly='viewFlag' ref="fileTables" 
				:url='fileDateUrl' 
				:default-checked="defaultChecked" 
				@row-click="rowClickUpgrade" 
				@load-success='loadSuccess' 
				:height="height" pagination="true" 
				:query-params="fileParams">
				<el-table-column label='<%=rb.getString("XuanZe")%>' width="60" >
					<template slot-scope="scope" >
	              		<div class="tableSelectCls">
							<span class='tableDiv el-icon el-icon-status-yes selected-status'></span>
						</div>
	            	</template>
				</el-table-column>

				<el-table-column key="file_name" v-if="ruleForm.upgradeMode == 'image'" prop="file_name" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>"  width="300"></el-table-column>
				<el-table-column key="fileName"  v-if="ruleForm.upgradeMode == 'module'" prop="fileName" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>"  width="300"></el-table-column>

				<el-table-column key="version" prop="version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>" width="250"></el-table-column>

				<el-table-column key="model_name" v-if="ruleForm.upgradeMode == 'image'" prop="model_name" show-overflow-tooltip label="<%=rb.getString("ChanPinXingHao")%>" width="160" ></el-table-column>
				<el-table-column key="size" v-if="ruleForm.upgradeMode == 'image'" prop="size" label="<%=rb.getString("WenJianDaXiao")%>" width="150"></el-table-column>
				<el-table-column key="upload_time" v-if="ruleForm.upgradeMode == 'image'" prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" width="180"></el-table-column>
				<el-table-column key="moduleVersion" v-if="ruleForm.upgradeMode == 'image' && ruleForm.needModuleUpgrade == 'true'" prop="moduleVersion" show-overflow-tooltip label="<%=rb.getString("MoKuaiBanNen")%>"></el-table-column>
				<el-table-column key="desc" v-if="ruleForm.upgradeMode == 'image'" prop="desc" show-overflow-tooltip label="<%=rb.getString("MiaoShu")%>"></el-table-column>

				<el-table-column key="moduleName" v-if="ruleForm.upgradeMode == 'module'" prop="moduleName" show-overflow-tooltip label="<%=rb.getString("MoKuaiMingCheng")%>" min-width="110" ></el-table-column>
				<el-table-column key="destVersion" v-if="ruleForm.upgradeMode == 'module'" prop="destVersion" show-overflow-tooltip label="<%=rb.getString("MuBiaoBanBen")%>" width="120"></el-table-column>
				<el-table-column key="fileSize" v-if="ruleForm.upgradeMode == 'module'" prop="fileSize" label="<%=rb.getString("WenJianDaXiao")%>" width="150"></el-table-column>
				<el-table-column key="uploadTime" v-if="ruleForm.upgradeMode == 'module'" prop="uploadTime" label="<%=rb.getString("ShangChuanShiJian")%>" width="180"></el-table-column>
				<el-table-column key="description" v-if="ruleForm.upgradeMode == 'module'" prop="description" show-overflow-tooltip label="<%=rb.getString("MiaoShu")%>"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='fileIds' style='margin-left:45px;'>
			<el-input v-model='ruleForm.fileIds' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:60px;border:none;margin-left:30px;margin-top:20px;display:flex;'>
			<el-form-item  prop='status'>
				<el-radio-group v-model="ruleForm.status" :disabled='viewFlag'>
					<el-radio border label="active" style='margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<!--
					<el-radio border label="online"><%=rb.getString("ShangXianZhiXing")%></el-radio>
					-->
					<el-radio border label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;' >
				<el-date-picker style='vertical-align:middle;' value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="viewFlag||setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
		
		<div style='width:99%;height:1px;background:#E9E9E9;margin-bottom:30px;'></div>
		<!-- 接口还未调测 执行策略 -->
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("RenWuShuXing")%></span>
		</div>
		<el-form-item style='display:inline-flex;margin:10px 45px 20px;width:50%;' label="<%=rb.getString("LiXianSheBei")%>">
			<el-checkbox v-model="ruleForm.isOnlineExecute" true-label="1" false-label="0" :disabled='viewFlag'></el-checkbox> <%=rb.getString("DengDaiShangXianChongShi")%>
		</el-form-item>
		
		<el-form-item style='margin:20px 45px;' :label-width="maxNumLabelWidth" prop='maxConcurrentNumber' label="<%=rb.getString("SheBeiBingFaShu")%>">
			<el-input-number @change="maxConcurrentNumberChange" v-model='ruleForm.maxConcurrentNumber' :min="5" :max="100" :disabled='viewFlag' style="width:200px;height:28px;line-height:28px;"></el-input-number>
		</el-form-item>
	</el-form>
	<el-dialog class='dialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="left">
			<div>
				<el-form-item  prop='type' label="Input Type" style='margin-bottom:10px;'>
					<el-radio-group v-model="addListForm.type" style="padding-top:13px;">
						<el-radio label="mac" style='margin-right:30px;'>MAC</el-radio>
						<el-radio label="sn" style='margin-bottom:0px;'><%=rb.getString("CPEBianMa")%></el-radio>
					</el-radio-group>
				</el-form-item>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
new Vue({
	el:'#cpeUpgradeAddTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/task/upgrade/cpe/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskname.trim(),
					taskType:'1'
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(data["message"] == "true"){
							callback(new Error('<%=rb.getString("RenWuMingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}
				}).catch(function(error){
					callback()
				})
			}
		};
		var validateTime = (rule,value,callback) => {
			if(this.ruleForm.status !== 'timing'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		};
		var validateCellCodes = (rule,value,callback) => {
			if(this.ruleForm.deviceAssign == '1'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
				}else{
					callback();
				}
			}
		};
		 var validatorNum = (rule,value,callback) => {
				
				var serialNumber = value||'',
					list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;}),
					temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/,
					noColTemp = /^([A-Fa-f0-9]{2}){6}$/,
					tempSn = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
					/*snArr =  serialNumber.split(/[(\r\n)\r\n]+/g),
					list = [];
				
				if(snArr.length>1) {// 多行
					snArr.map(function(str){
						var item = str.trim(),
							lastIdx = item.lastIndexOf(';'),
							length = item.length-1;
						
						if(lastIdx>=0 && lastIdx == length) {
							list.push(item.substring(0,lastIdx));
						}else if(item) {
							list.push(item);
						}
					});
				}else {// 单行
					list = serialNumber.split(';');
				}
				
				//判断最后一项是否为空 为空删除
				if(list[list.length-1] == ""){
					list.splice(list.length-1)
				}*/
				
				
			    if(serialNumber != null && serialNumber.length != 0){
					var nameFlag;
					if(this.addListForm.type == 'mac'){
						nameFlag = list.every(function(item,index){
							return (temp.test(item) ||  noColTemp.test(item)) 
						})
					}else{
						nameFlag = list.every(function(item,index){
							return tempSn.test(item)
						})
					}
					
					if(nameFlag){
						callback()
					}else{
						if(this.addListForm.type == 'mac'){
							callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
						}else{
							callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
						}
					}
				}else if (serialNumber == null || serialNumber.length == 0) {
					if(this.addListForm.type == 'mac'){
						callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}else{
					callback();
				}
			};
		return {
			moduleNameList: [],
			deviceData:{}, // 新建任务接收的设备信息
			noFilterFile:true,
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			deviceGroups:[],//高级查询设备组选择下拉内容
			versions:[],//高级查询版本选择下拉内容
			modelName:[], // 高级查询model下拉内容
			fileParams:{
				timeZone:timeZone,
				isShowSlave:false,
				fileId:'',
				model_name:'',
				moduleName: ''
			},
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			deviceSelect:[],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			ruleForm:{
				taskname:'${addTaskName}',
				upgradeMode: 'image',
				moduleName: '',
				needModuleUpgrade: 'false',
				cellCodes:'',
				fileIds:'',
				fileName:'',
				status:'active',
				exetime:'',
				deviceAssign:'0',
				version:'',
				connection_status: '',
				maxConcurrentNumber: 20,
				isOnlineExecute: '0'
			},
			rules:{
				taskname:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{validator:validateCellCodes,trigger:'change'}
				],
				fileIds:[
					{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			queryParams:{
				moduleName: '',
				searchText:'',
				serial_number :'',
				imsi:'',
				host_name :'', 
				group_id :'', 
				software_version:'',
				model_name:'',
				cell_name: '',
				pci:'',
				connection_status: ''
			},
			queryForm:{
				serial_number :'', //高级查询基站编码
				imsi:'',
				host_name :'', //高级查询基站名称
				group_id :[], //高级查询选择的设备组
				software_version:[],//高级查询选中的版本 
				model_name:[],//高级查询选中的model 
				cell_name: '',
				pci:'',
				connection_status: ''
			},
			taskId:'',
			productValue:'',
			operationType:'',
			defaultTaskName:'',
			showPairGrid:true,
			fileTableUrl:'',
			fileDateUrl:'',
			fileVersion:'',
			viewFlag:false,
			productModelDifference:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.getNow()-8.64e7;
				}
			},
			addListForm:{
				serialNumber:'',
				type:'mac'
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:"<%=rb.getString("TianJia")%>",
			onlineOptions:[
				{label:"<%=rb.getString("QuanBu")%>",value:""},
				{label:"<%=rb.getString("ZaiXian")%>",value:"1"},
				{label:"<%=rb.getString("LiXian")%>",value:"0"}
			],
		}
	},
	computed:{
		maxNumLabelWidth(){
			return isLocalZH == true ? '200px' : '290px';
		},
	},
	methods:{ 
		/**
		 * 初始化
		 * @param id:number     cpe升级任务id
		 * @param type:string   任务类型  'addTask' -- 新增  'taskView'-- 查看  'taskEdit' -- 修改
		 * @param productVal:string     设备类型  '1': ODU  '2': IDU
		 * @param deviceData:object  新建升级任务传递的设备数据  
		*/
		init(id,type,productVal,deviceSelect){ 
			var vm = this;
			vm.taskId = id ;
			vm.operationType = type;
			vm.deviceSelect = deviceSelect;
			//vm.oldProductValue = productVal;
			//vm.productValue = isJumpToPage ? isJumpToPage.file_type : productVal;
			if(type == 'taskView'){
				vm.viewFlag = true;
				vm.showPairGrid = false;
			}else if(type == 'addTask'){
				if(deviceSelect.length > 0){
					vm.$refs.cpeDevicePairgrid.appendCheckedRows(deviceSelect);
				}
			}
			
			vm.getAdvanceCntent();
			vm.getModuleNames();

			if(vm.ruleForm.upgradeMode == 'image') {
				vm.fileTableUrl = '${ctx}/cell/version/queryfileInfosList.action?file_type=9&timeZone='+timeZone;
			}else {
				vm.fileTableUrl = '${ctx}/cell/version/queryModuleFileInfos.action';
			}

			//vm.leftUrl="${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect="+vm.productValue;
			vm.leftUrl="${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action";

			if(type !== "addTask"){
				vm.$nextTick(function(){
					vm.rightUrl="${ctx}/task/upgrade/cpe/getTaskSelectedList.action?taskId="+vm.taskId+"&timeZone="+timeZone;
				})
				vm.getTaskDateInfo();
			}else{
				vm.fileDateUrl = vm.fileTableUrl;
			}
			initForm(vm.$refs.ruleForm);
		},
		getModuleNames() {
			var vm = this;

			axios.post('${ctx}/cell/version/getModuleNames.action').then(function(res){
				var data = res.data || [];

				vm.moduleNameList = data;
			});
		},
		//获取高级查询下拉列表内容
		getAdvanceCntent(){
			var vm = this;
			axios.post('${ctx}/cell/CPE/getCpeSelectFilter.action',stringify({
				forSelect:vm.productValue,
				selectType:"deviceGroup"
			})).then(function(response){
				vm.deviceGroups = response.data || [];
			}).catch(function(error){});
			axios.post('${ctx}/cell/CPE/getCpeSelectFilter.action',stringify({
				forSelect:vm.productValue,
				selectType:"version"
			})).then(function(response){
				vm.versions = response.data || [];
			}).catch(function(error){})
			axios.post('${ctx}/cell/CPE/queryModelNames.action',stringify({
				forSelect:vm.productValue
			})).then(function(response){
				vm.modelName = response.data || [];
				if(vm.modelName.length > 0 ){
					//vm.queryParams.model_name = vm.modelName[0].model_name;
					//vm.queryForm.model_name = vm.modelName[0].model_name;
				}
			}).catch(function(error){})
		
		},
		// 设备执行类别  1 全部执行 2 指定执行
		deviceAssignChange(val){
			var vm = this;

			if(val == '1'){
				vm.rightUrl = '';
			}else{
				vm.rightUrl="${ctx}/task/upgrade/cpe/getTaskSelectedList.action?taskId="+vm.taskId+"&timeZone="+timeZone;
			}
		},
		connectionStatusChange(val){
			var vm = this;
			vm.queryParams.connection_status = val;			
		},
		// Upgrade Mode 切换事件
		modeChange(val) {
			var vm = this;

			vm.ruleForm.moduleName = '';
			vm.rowData = {},
			vm.ruleForm.fileIds = '';
			vm.ruleForm.fileName = '';
			vm.moduleChange('');

			vm.$refs.fileTables.setCurrentRow();
		},
		moduleChange(val) {
			var vm = this;

			vm.leftUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?moduleName=" + val;

			vm.queryParams.moduleName = val;
			vm.fileParams.moduleName = val;

			if(vm.ruleForm.upgradeMode == 'image') {
				vm.fileTableUrl = '${ctx}/cell/version/queryfileInfosList.action?file_type=9&timeZone='+timeZone;
			}else {
				vm.fileTableUrl = '${ctx}/cell/version/queryModuleFileInfos.action';
			}
			vm.fileDateUrl = vm.fileTableUrl;
		},
		// 文件过滤开关
		filterFileChange(val){
			var vm = this;

			if(val){
				vm.fileParams.model_name = vm.deviceData.modelName;
			}else{
				vm.fileParams.model_name = '';
			}
		},
		// 文件过滤开关
		filterModuleFileChange(val){
			var vm = this;

			if(val){
				vm.fileParams.module_name = vm.deviceData.moduleName;
			}else{
				vm.fileParams.module_name = '';
			}
		},
		// 模糊查询
		query(val){
			var vm = this;
			//vm.resetQuery();
			vm.advanceQuery();
			vm.queryParams.searchText = val;
		},
		// 高级查询
		advanceQuery(){
			var vm = this;
			//this.queryParams.searchText = "";
			var arrGroup = this.queryForm.group_id;
			var resGroup = arrGroup.indexOf("");
			var arrVer = this.queryForm.software_version;
			var resVer = arrVer.indexOf("");
			var arrModel = this.queryForm.model_name;
			var resModel = arrModel.indexOf("");
			
			this.queryParams.group_id = resGroup == -1 ? arrGroup.join(",") : '';
			this.queryParams.software_version = resVer == -1 ? arrVer.join(",") : '';
			this.queryParams.model_name = resModel == -1 ? arrModel.join(",") : '';
			this.queryParams.serial_number = this.queryForm.serial_number;
			this.queryParams.host_name = this.queryForm.host_name;
			
			this.queryParams.cell_name = this.queryForm.cell_name;
			this.queryParams.imsi = this.queryForm.imsi;
			this.queryParams.pci = this.queryForm.pci;
			this.queryParams.connection_status = this.queryForm.connection_status
			
			//Object.assign(vm.queryParams, vm.queryForm);
		},
		// 高级查询重置
		resetQuery(){
			var vm = this,
				params = {
					searchText: '',
					serial_number : '', //高级查询基站编码
					imsi:'',
					host_name :"", //高级查询基站名称
					group_id :[], //高级查询选择的设备组
					software_version:[],//高级查询选中的版本 
					model_name:[],//高级查询选中的model 
					cell_name: "",
					pci:'',
					connection_status: ''
				};
			
			Object.assign(vm.queryForm, params);
			Object.assign(vm.queryParams, params);
		},
		// cpe设备选择事件
		selectChange(selection){
			var vm = this;
			vm.selection = selection;
			
		},
		// 升级文件选择事件
		rowClickUpgrade(row){
	        this.rowData = row;
	    },
		// 获取cpe升级任务详情
		getTaskDateInfo(){
			var vm = this,
				params={
					taskId:vm.taskId,
					timeZone:timeZone
				};
			
			axios.post('${ctx}/task/upgrade/cpe/getTask.action',stringify(params)).then(function(response){
				let data = response.data;
				vm.ruleForm.taskname = data.TASK_NAME;
				vm.ruleForm.fileIds = data.FILE_ID;
				vm.ruleForm.status = data.CREATE_STATUS;
				vm.ruleForm.upgradeMode = data.upgradeMode;
				vm.ruleForm.moduleName = data.moduleName;
				vm.ruleForm.needModuleUpgrade = data.needModuleUpgrade;
				vm.ruleForm.maxConcurrentNumber = data.maxConcurrentNumber;
				vm.ruleForm.isOnlineExecute = data.isOnlineExecute;

				//vm.productValue = data.PRODUCT;
    			//vm.oldProductValue = data.PRODUCT;
				vm.ruleForm.deviceAssign = data.DEVICE_ASSIGN;
				if(data.CREATE_STATUS == 'timing'){
					vm.ruleForm.exetime = data.START_TIME;
				}
				if(data.PRODUCT == "1"){
					vm.leftUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action";
					if(vm.operationType !== 'taskView'){
						vm.fileTableUrl = "${ctx}/cell/version/queryfileInfosList.action?timeZone="+timeZone;
					}else{
						vm.fileTableUrl = "${ctx}/cell/version/queryfileInfosList.action?fileId="+data.FILE_ID+"&timeZone="+timeZone;
					}
				}else{
					vm.leftUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action";
					if(vm.operationType !== 'taskView'){
						vm.fileTableUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=9"+"&timeZone="+timeZone;
						if(vm.ruleForm.upgradeMode == 'module') {
							vm.fileTableUrl = "${ctx}/cell/version/queryModuleFileInfos.action?timeZone="+timeZone;
						}
					}else{
						vm.fileTableUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=9"+"&fileId="+data.FILE_ID+"&timeZone="+timeZone;
						if(vm.ruleForm.upgradeMode == 'module') {
							vm.fileTableUrl = "${ctx}/cell/version/queryModuleFileInfos.action?fileId="+data.FILE_ID+"&timeZone="+timeZone;
						}
					}
				}
				vm.defaultChecked = [data.FILE_ID+''];
				vm.defaultTaskName = data.TASK_NAME;
				vm.fileDateUrl = vm.fileTableUrl;
				
			}).catch(function(error){})
		},
		// 提交
		submit(){
	    	var vm = this,
                message = '<%=rb.getString("ChengGong")%>';
            // 防止多次提交
            if(cpeUpgrade.slideSubmitLoading)return

			if(vm.operationType == 'taskEdit'){
				if(!isFormChanged(vm.$refs.ruleForm)){
					vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						type:'warning'
					}).then().catch();
					return;
				}
			}
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {},urls='';
	    			params.timeZone = timeZone;
	    			params.cellCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskname;
					//params.productValue = vm.productValue;
					params.deviceAssign = vm.ruleForm.deviceAssign;
					params.version = vm.ruleForm.version;
					params.taskType = '1';
					params.rawMode = 'false';
	    			params.status = vm.ruleForm.status;

	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
	    			params.file_id = vm.ruleForm.fileIds;
					params.file_name = vm.ruleForm.fileName;

					params.upgradeMode = vm.ruleForm.upgradeMode;
					params.moduleName = vm.ruleForm.moduleName;
					params.needModuleUpgrade = vm.ruleForm.needModuleUpgrade;
	    			params.maxConcurrentNumber = vm.ruleForm.maxConcurrentNumber;
	    			params.isOnlineExecute = vm.ruleForm.isOnlineExecute;

					if(vm.operationType == 'addTask'){
						urls = '${ctx}/task/upgrade/cpe/addTask.action';
					}else{
						params.taskId = vm.taskId
						urls = '${ctx}/task/upgrade/cpe/updateTask.action';
					}
					axios.post('${ctx}/task/upgrade/cpe/isNeedMoreTimeForUpgrade.action', stringify({
       						cpeCodes:params.cellCodes,
							//productValue:vm.productValue,
							deviceAssign:vm.ruleForm.deviceAssign,
       						destVersion:params.version
       					})).then(function(response){
       						let data = response.data;
       						
       						if (data["isNeedMoreTime"]) {
       							vm.$confirm( '<%=rb.getString("ShengJiShiJianJiaoChangShiFouJiXu")%>' ,'<%=rb.getString("QueRen")%>',{
       			    				customClass:'warningConfirm',
       								confirmButtonText:'<%=rb.getString("QueDing")%>',
       								cancelButtonText:'<%=rb.getString("QuXiao")%>',
       								type:'warning',
       								closeOnClickModal:false
       							}).then(function(){
       								vm.saveTask(urls,params,message);
       							}).catch()
       						} else {
       							if(vm.operationType == 'addTask'){
									   var msg = ''
									   if(vm.productModelDifference){
											msg = '<%=rb.getString("PiPeiXingHaoWenJianQueDing")%>' +'<%=rb.getString("QueRenXinJianRenWu")%>';
									   }else{
										   	msg = '<%=rb.getString("QueRenXinJianRenWu")%>';
									   }
       								vm.$confirm( msg ,'<%=rb.getString("QueRen")%>',{
       				    				customClass:'warningConfirm',
       									confirmButtonText:'<%=rb.getString("QueDing")%>',
       									cancelButtonText:'<%=rb.getString("QuXiao")%>',
       									type:'warning',
       									closeOnClickModal:false
       								}).then(function(){
       									vm.saveTask(urls,params,message);	
       								}).catch()
       							}else{
       								vm.saveTask(urls,params,message);	
       							}
       						}
          				}).catch(function(error){
          				
          				})
					
	    			
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(urls,params,message){
			var vm = this;
            cpeUpgrade.slideSubmitLoading = true;
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    cpeUpgrade.$refs.cpeUpgradedeviceTable.clearSelection();
                    eventBus.$emit('hide-cpeUpgrade-slide');
                    isJumpToPage = '';
				}else{
					vm.$message.error(data["message"]);
                    cpeUpgrade.slideSubmitLoading = false;
				}
			}).catch(function(error){})
			
		},
		// 取消
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
			if(vm.operationType !== 'taskView'){
				if(isFormChanged(this.$refs.ruleForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-cpeUpgrade-slide');
						isJumpToPage = ''
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-cpeUpgrade-slide')
					isJumpToPage = ''
				}
			}else{
				eventBus.$emit('hide-cpeUpgrade-slide');
				isJumpToPage = ''
			}
			
		},
		// cpe 升级文件表格加载成功回调
		loadSuccess(){
			var vm = this;
			var tb = vm.$refs.fileTables;
			var rows = tb.tbData;
			if(vm.operationType !== 'addTask'){
				vm.updateRow(tb.tbData,tb);
			}else{
				if(isJumpToPage){
					rows.map(function(item){
						if(item.id == isJumpToPage.vid){
							tb.setCurrentRow(item);
							vm.ruleForm.fileIds = isJumpToPage.vid;
						}
					})
				}
			}
			
		},
		updateRow(rows,tb){
			var vm = this;
			axios.post('${ctx}/task/upgrade/cpe/getTask.action',stringify({
				taskId : vm.taskId,
				timeZone : timeZone
			})).then(function(response){
				var data = response.data;
				rows.map(function(item){
					if(item.id == data.FILE_ID){
						tb.setCurrentRow(item)
					}
				})
			})
		},
		// 定时时间失焦事件
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime');
		},
		// 产品类型格式化
		productFmt(row,column,value,index){
			if(value == 'CPE_VERSION' || value == "ODU"){
				return "ODU";
			}else if(value == 'CPE_IDU_VERSION' || value == "IDU"){
				return "IDU";
			}
		},
		addBatchSn(){
			this.listVisible = true;
		},
		closeBatchSn(){
			this.listVisible = false;
			this.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '',
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			//snStr = snStr.replace(/[(;\s*)(\r\n)\r\n]+/g,';');
			
			var params = {
				type:vm.addListForm.type
			};
			if(vm.addListForm.type == 'mac'){
				params.macs = list.join(';');
			}else{
				params.sns = list.join(';');
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/CPE/queryCpeInfoByMacs.action",stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){		
							vm.$refs.cpeDevicePairgrid.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message("<%=rb.getString("MeiYouKePiPeiSheBei")%>")
						}
					})
				}
			})
		},
		// 最大并发数失焦事件
		maxConcurrentNumberChange(event){
			var vm = this,
				value = vm.ruleForm.maxConcurrentNumber;
			if(value == undefined || value == '' || value == null){
				vm.$nextTick(()=>{
					vm.ruleForm.maxConcurrentNumber = 5;
				})
			}
		},
	},
	watch:{
		rowData(newVal){
			var vm = this;

			this.ruleForm.fileIds = newVal.id;
			this.ruleForm.fileName = vm.ruleForm.upgradeMode == 'image'?newVal.file_name:newVal.fileName;
			this.ruleForm.version = newVal.version;
			if(this.deviceData.modelName){
				var modelNames= '',deviceModelNameList=[];

				modelNames = this.deviceData.modelName;
				deviceModelNameList = modelNames.split(',');
				fileModelNameList = newVal.model_name ? newVal.model_name.split(',') : [];
				var arr = [];
				deviceModelNameList.map((item)=>{
					if(fileModelNameList.indexOf(item) == -1){
						arr.push(item)
					}
				})
				if(arr.length>0 ){
					if(vm.ruleForm.upgradeMode == 'image') {
						vm.productModelDifference = true // 等于true的时候表示数组里不止有指定的  弹出提示
					}else {
						vm.productModelDifference = false
					}
				}else{
					vm.productModelDifference = false // 等于false的时候数组里只有指定的 不弹出提示
				}
			}
		},
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		selection(){
			var vm = this,
				data = this.$refs.cpeDevicePairgrid.getData();
				cellCodes = '',
				modelNames = '' ,
				cellCodeList = [],
				modelNameList = [];

			if(data.length != 0){
				data.map(function(item){
					cellCodeList.push(item.small_cell_code);
					if(item.model_name){
						modelNameList.push(item.model_name);
					}
				})
			}
			cellCodes = cellCodeList.join(',');
			modelNames = modelNameList.join(',');
			vm.ruleForm.cellCodes = cellCodes;
			vm.deviceData.modelName = modelNames;

			if(vm.noFilterFile){
				vm.fileParams.model_name = vm.deviceData.modelName;
			}
		}
	},
	mounted(){
		eventBus.$off('cpe-upgrade-taskInit').$on('cpe-upgrade-taskInit',this.init);
		eventBus.$off('cpe-upgrade-addSubmit').$on('cpe-upgrade-addSubmit',this.submit);
		eventBus.$off('cpe-upgrade-addCancel').$on('cpe-upgrade-addCancel',this.cancel);
	}
})
</script>