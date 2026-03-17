<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.kpiTemplateAddModifyInfo {
		width: 100%;
		height: 100%;
		position: relative;
		overflow: hidden;
	}
	.kpiTemplateAddModifyInfo .fileContent .el-form-item__content { 
		display: flex;
	}
	.kpiTemplateAddModifyInfo .footer{
		width:100%;
		border-top:1px solid #D5DCEC;
		position:absolute;
		bottom:0px;
		height:50px;
		line-height:50px;
		background:#FFFFFF;
		z-index:99;	
	}
	.kpiTemplateAddModifyInfo .houseBox {
		display: flex;
	}
	.kpiTemplateAddModifyInfo .houseBox .el-form-item__label {
		line-height: 28px;
		margin-right: 24px;
	}
	.kpiTemplateAddModifyInfo .specilGroup {
		margin: 4px 14px 4px;
	}
	.kpiTemplateAddModifyInfo .specilGroup .el-checkbox-button__inner {
		border: 0 !important;
		padding: 0;
		width: 64px;
		height: 24px;
		line-height: 24px;
		margin: 5px 4px;
	}
	.kpiTemplateAddModifyInfo  .specilGroup .el-checkbox-button.is-checked .el-checkbox-button__inner{
		background-color:rgba(var(--main-color-rgba1),0.08) !important;
		border: 1px solid var(--main-color) !important;
		color: var(--main-color) !important;
		box-shadow: 0 0 0 0 var(--main-color) !important;
		border-radius: 4px;
	}
	.kpiTemplateAddModifyInfo .el-checkbox-button.is-checked .el-checkbox-button__inner{
		background-color:rgba(var(--main-color-rgba1),0.08) !important;
		border-color:var(--main-color) !important;
		color: var(--main-color) !important;
		box-shadow: -1px 0 0 0 var(--main-color) !important;
	}
	.kpiTemplateAddModifyInfo .specilGroup .el-checkbox-button {
		text-align: center;
	}
	.kpiTemplateAddModifyInfo .commonForm {
		width: 100%;
		height: calc(100% - 50px);
		overflow: auto;
	}
	.kpiTemplateAddModifyInfo .selectLimit { 
		color: #4D84FF; 
		margin-left: 5px;
	}
	.kpiTemplateAddModifyInfo .deviceListBox .el-pairgrid-title {
		top: 12px;
		right: 10px !important;
	}
	.kpiTemplateAddModifyInfo .deviceListBox .pairgrid-right {
		top: 52px !important;
	}
	.kpiTemplateAddModifyInfo .el-icon-operation-date:before, 
	.kpiTemplateAddModifyInfo .el-icon-operation-time:before{
		color: #4D84FF !important; 
	}
	#kpiViewTemplatePage .specilBusyTime .el-radio-button__inner {
		border: 0;
		padding: 8px 0;
		width: 110px;
	}
	#kpiViewTemplatePage .specilBusyTime .el-radio-group,
	#kpiViewTemplatePage .specilBusyTime .el-radio-button {
		display: block;
	}
	#kpiViewTemplatePage .editButton{
		position: absolute;
		right: 130px;
		top: 10px;
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
	}
	#kpiViewTemplatePage .editButton i{
		font-size:14px !important;
	}
	#kpiViewTemplatePage .editButton span{
		font-size:12px;
	}
	#kpiViewTemplatePage .basicLeftLabel { 
		display: flex; 
	}
	#kpiViewTemplatePage .basicLeftLabel .el-form-item__label { 
		margin-right: 16px;
		line-height: 28px;
	}
	#kpiViewTemplatePage .productTypeSelect .el-input__inner {
		height: 26px;
		line-height: 26px;
	}
	#kpiQuery .queryGroup {height: 26px;}
	#kpiViewTemplatePage .queryGroup .el-input__inner { width: 200px; height: 26px; line-height: 26px; padding: 0; }
	#kpiViewTemplatePage .el-cascader {
		line-height: 26px;
	}
	#kpiViewTemplatePage .el-cascader .el-input {
		width: 300px;
	}
</style>
<!--kpi view, 新建，修改，查看模板页面  -->
<div id="kpiViewTemplatePage" class="flex-ctn kpiTemplateAddModifyInfo">

	<el-form ref="addModifyViewForm" :model="addModifyViewForm" :rules="formRules" label-position="top" class='commonForm' :disabled='commonOperType == "view"'>
		<div class="commonItemBox">  
			<div class='commonLeftRight'>
				<div class="itemTitle"><%=rb.getString("JiBenXinXi")%></div>
		  		<div class='itemBox'>
		  			<el-form-item prop="tempName" label='<%=rb.getString("MuBanMingCheng")%>'>
						<el-input v-model="addModifyViewForm.tempName" maxLength="50" style='width: 320px;' :disabled='nameDescDisabled'></el-input>
						<!--内置模板不能设置为私有模板(Basic,All Operator)-->
						<el-checkbox v-model="addModifyViewForm.isPublic" true-label="1" false-label="0" style='margin: 0 10px 0 20px;' :disabled='nameDescDisabled'></el-checkbox>	
						<span class='commonSize14'>Set as public template</span>
						<span class='commonNotes12' style="margin-left: 5px;"><%=rb.getString("GongGongMuBanTiShi")%></span>
					</el-form-item>

					<!--详情才有这俩字段-->
					<el-form-item prop="creator" label='<%=rb.getString("KPIChuangJianRen")%>' v-if='commonOperType == "view"'>
						<el-input v-model="addModifyViewForm.creator" maxLength="50" style='width: 320px;' :disabled='nameDescDisabled'></el-input>
					</el-form-item>
					<el-form-item prop="updator" label='<%=rb.getString("KPIGengXinRen")%>' v-if='commonOperType == "view"'>
						<el-input v-model="addModifyViewForm.updator" maxLength="50" style='width: 320px;' :disabled='nameDescDisabled'></el-input>
					</el-form-item>

					<el-form-item prop="description" label='<%=rb.getString("MiaoShu")%>'>
						<el-input type="textarea" v-model="addModifyViewForm.description" maxLength="500" style="width:800px;" :disabled='nameDescDisabled'></el-input>
					</el-form-item>
		  		</div>
			</div>
	   	</div>
	   	<div class="commonItemBox">
			<div class='commonLeftRight'>
				<div class="itemTitle"><%=rb.getString("ZhouQiSheDing")%></div>
		  		<div class='itemBox'>
		  			<el-form-item label='<%=rb.getString("ChaXunLiDu") %>' prop="reportPeriod">
		                <el-radio-group v-model="addModifyViewForm.reportPeriod">
		                    <el-radio label="15" border>15Min</el-radio>
	                    	<el-radio label="60" border style='margin-left: 20px;'>60Min</el-radio>
	                    	<el-radio label="1440" border style='margin-left: 20px;'>24hour</el-radio>
		                </el-radio-group>
		            </el-form-item>
		            <!-- 选中： 所有时段； 未选中： 指定时段 ; 周期粒度切换  （当选择24小时粒度时，隐藏定制时段相关显示 ）; 切换所有时段或是定制时段 （ 当前选择所有时段时，定制时间置灰不可点击 ）
		            	所有时段：week, hour 参数为空；24小时粒度：week, hour 参数为空；
		            	// 周期粒度切换  （当选择24小时粒度时，隐藏定制时段相关显示 ）置灰
		            	//切换所有时段或是定制时段 （ 当前选择所有时段时，定制时间置灰不可点击 ）-->
		            <el-form-item prop="checkAll" label='<%=rb.getString("XiaoShi")%>' class='houseBox'>
						<el-checkbox v-model="addModifyViewForm.checkAll" true-label="1" false-label="0" :disabled='addModifyViewForm.reportPeriod == "1440"'></el-checkbox>
				        <span class='commonColor3 commonFontSize12' style='margin-left: 6px;'><%=rb.getString("SuoYouAny") %></span>
					</el-form-item>
					<div class='commonWarp' style='width: 620px;'>
						<div class='commonFlex' style='border-bottom: 1px solid #D5DCEC;height: 60px;'>
							<div style='width: 130px;' class='commonFlex'>
								<span class="el-icon el-icon-operation-date commonIcon" style='margin: 15px 10px 0 12px; background: #F2F6FF;'></span>
								<span class='commonSize14' style='font-weight: bold; margin-right: 30px; line-height: 60px'><%=rb.getString("Zhou")%></span>
							</div>
		                    <el-form-item label=" " prop="week" style='margin-top: 10px;'>
				              	<el-checkbox-group v-model="addModifyViewForm.week" class='normalGroup'>
	                        		<el-checkbox-button v-for="item in weekArr" :label="item.value" border :disabled='addModifyViewForm.checkAll == "1" || addModifyViewForm.reportPeriod == "1440"'>{{item.label}}</el-checkbox-button>
			                    </el-checkbox-group>
				            </el-form-item>
						</div>
						<div class='commonFlex'>
							<div style='width: 130px;'>
								<div class="commonFlex">
									<span class="el-icon el-icon-operation-time commonIcon" style='margin: 16px 10px 0 12px; background: #F2F6FF;'></span>
									<span class='commonSize14' style='font-weight: bold; margin-right: 30px; line-height: 54px'><%=rb.getString("XiaoShi")%></span>
								</div>
								<el-form-item label="" prop="busyTime" class='specilBusyTime' style="display:inline-block; margin: 5px 10px;">
									<el-radio-group v-model="addModifyViewForm.busyTime">
										<el-radio-button label="6" :disabled='addModifyViewForm.checkAll == "1" || addModifyViewForm.reportPeriod == "1440"'><%=rb.getString("6MangShi")%></el-radio-button>
										<el-radio-button label="8" :disabled='addModifyViewForm.checkAll == "1" || addModifyViewForm.reportPeriod == "1440"'><%=rb.getString("8MangShi")%></el-radio-button>
									</el-radio-group>
								</el-form-item>
							</div>
							
		                    <el-form-item label=" " prop="hour" style='margin: 15px 0 20px; width: 462px; border: 1px solid #DFE2EE; border-radius: 4px;'>
				              	<el-checkbox-group v-model="addModifyViewForm.hour" @change="" class='specilGroup'>
	                        		<el-checkbox-button v-for="item in hourArr" :label="item.value" :disabled='addModifyViewForm.checkAll == "1" || addModifyViewForm.reportPeriod == "1440"'>{{item.label}}</el-checkbox-button>
			                    </el-checkbox-group>
				            </el-form-item>
						</div>
					</div>
		  		</div>
		  	</div>
		</div>
		<!-- 设备，设备组； 修改模板时： isAllOperator == 1，不显示此模块 -->
		<div class="commonItemBox" v-show='isAllOperator == false'>
			<div class='commonLeftRight'>
				<div class="itemTitle"><%=rb.getString("SheBeiLieBiao")%></div>
		  		<div class='itemBox'>
		  			<div style="display: flex;flex-direction: column;">
						<el-form-item label='<%=rb.getString("SheBeiXuanZheFangShi")%>' prop="selDeviceType" class='basicLeftLabel'>
							<el-radio-group v-model="addModifyViewForm.selDeviceType">
								<el-radio label="1" border><%=rb.getString("SheBeiZu")%></el-radio>
								<el-radio label="2" border style='margin-left: 20px;'><%=rb.getString("KPISheBei")%></el-radio>
							</el-radio-group>							
							<span class="selectLimit" v-show='commonOperType != "view"'>(<%=rb.getString("ZuiDuoXuanZe")%>  {{deviceNumLimit}} <%=rb.getString("ZuiDuoXuanZeDevice")%>)</span>
						</el-form-item>
						<div style="width: 100%; margin-top: 10px;">
				   			<div class="selectListCont">
								<div v-show='commonOperType == "view"'>
									<el-ctable v-if='addModifyViewForm.selDeviceType == "1"' style="width:100%; height: 260px; border:1px solid #DFE2EE;" :id="'group_list'" :pagination="false" ref="viewDevices" :url="viewServiceGroupsUrl" :row-key="'id'" >
										<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name"></el-table-column>
									</el-ctable>
									<el-ctable v-if='addModifyViewForm.selDeviceType == "2"' style="width:100%; height: 260px;border:1px solid #DFE2EE;" :id="'group_list'" :show-pager="true" ref="viewDevices" :url="viewServicesUrl" :row-key="'smallCellCode'" >
										<el-table-column v-if="templateNetType == 'enb' || templateNetType == 'gnb'" prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
										<el-table-column v-if="templateNetType == 'enb'" prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
										<el-table-column v-if="templateNetType == 'gnb'" prop='hostName' label='<%=rb.getString("GNBMingCheng")%>'></el-table-column>
										<el-table-column v-if="templateNetType == 'egw'" prop='serialNumber' label='<%=rb.getString("eGWBianMa")%>'></el-table-column>
										<el-table-column v-if="templateNetType == 'egw'" prop='hostName' label='<%=rb.getString("eGWMingCheng")%>'></el-table-column>
										<el-table-column v-if="templateNetType == 'enb'" prop='product' label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
									</el-ctable>
								</div>

								<div v-show='showDeviceGroup && commonOperType != "view" && templateNetType == "enb" ' class='deviceListBox'>
									<el-pairgrid :id="'selected_device_list'" ref="groupTable" :row-key="'id'"
										:rownumber="true"
										:right-url="groupRightUrl"
										:left-url="groupUrl"
										:height="'260px'"
										:row-key="'smallCellCode'"
										:query-params="deviceGroupForm"
										:title="deviceTitle"
										:limit="deviceNumLimit"
										:messages="commonMessage"
										@checked-change='groupChange'>
										<template slot="left">
											<el-table-column type="selection" :reserve-selection="true"></el-table-column>
											<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name"></el-table-column>
										</template>
										<template slot='toolbar'>
											<div class='commonFlex'>
												<div class="queryGroup">
													<el-input class='pairgrid-query' v-model="deviceGroupSearchText" @keyup.enter.native="deviceGroupQuery" placeholder='<%=rb.getString("SheBeiZuMingCheng")%>'></el-input>
											    	<i @click='deviceGroupQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
												</div>
												<el-form-item prop='device_type' style="margin: 0;"> 
													<el-radio-group v-model="addModifyViewForm.device_type" @change="deviceTypeChange" size="mini">
														<el-radio-button label="ENB,GSM"><%=rb.getString("QuanBuSheBei")%></el-radio-button>
														<el-radio-button label="ENB"><%=rb.getString("4GSheBei")%></el-radio-button>
														<el-radio-button label="GSM"><%=rb.getString("2GSheBei")%></el-radio-button>
													</el-radio-group>
												</el-form-item>
											</div>
										</template>
										<template slot='right'>
											<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name"></el-table-column>
										</template>
									</el-pairgrid>
									<el-form-item prop='autoAccessDeviceGroupId' style="margin: 0;" label-width="0">
										<el-input v-model='addModifyViewForm.autoAccessDeviceGroupId' v-show="false"></el-input>
									</el-form-item>
								</div>
								<!-- 设备组列表 -->
								<div v-show='showDeviceGroup && commonOperType != "view" && templateNetType != "enb" ' class='deviceListBox'>
									<el-ctable :id="'selected_device_list'"
										ref="groupTable"
										:row-key="'id'"
										:url="groupUrl"
										:height="'260px'"
										:pagination="false"
										:rownumber="true"
										@selection-change="groupChange"
										style="border:1px solid #DFE2EE;">
										<el-table-column label='' type="selection" :reserve-selection="true"></el-table-column>
										<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name"></el-table-column>
									</el-ctable>
									<el-form-item prop='autoAccessDeviceGroupId' style="margin: 0;" label-width="0">
										<el-input v-model='addModifyViewForm.autoAccessDeviceGroupId' v-show="false"></el-input>
									</el-form-item>
								</div>

								<!-- 设备列表 -->
								<div v-show='!showDeviceGroup && commonOperType != "view"' class='deviceListBox'>
									<el-pairgrid :id="'select_device_list'"
										:rownumber="true"
										ref="cpairgrid"
										:right-url="rightUrl"
										:left-url="leftUrl"
										:height="'260px'"
										:row-key="'smallCellCode'"
										:query-params="queryForm"
										:title="deviceTitle"
										:limit="deviceNumLimit"
										:messages="commonMessage"
										@checked-change='devicesChange'>
										<template slot="prev">
											<el-ctable style="width:300px;" :id="'group_list'" :show-pager="false" ref="group" :url="groupUrl"
												:height="'100%'" :show-header="false" :row-key="'id'" @current-change="queryGroupChange" :pagination="false">
												<template slot='toolbar'>
													<p style="margin-left: 20px;"><%=rb.getString("SheBeiZu")%></p>
												</template>
												<el-table-column label='<%=rb.getString("SheBeiZu")%>' prop="group_name"></el-table-column>
											</el-ctable>
										</template>
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
											<el-table-column v-if="templateNetType == 'enb' || templateNetType == 'gnb'" prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'enb'" prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'gnb'" prop='hostName' label='<%=rb.getString("GNBMingCheng")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'egw'" prop='serialNumber' label='<%=rb.getString("eGWBianMa")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'egw'" prop='hostName' label='<%=rb.getString("eGWMingCheng")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'enb'" prop='product' label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
										</template>
										<template slot='toolbar'>
											<div class='commonFlex'>
												<el-form-item label="" v-if="templateNetType == 'enb'" style=" margin: 0 0 0 20px;" class="commonFlex productTypeSelect">
													<el-select v-model="product_type" @change='productChange'>
														<el-option v-for="item in productTypeList" :key="item.value" :label="item.label" :value="item.value"></el-option>
													</el-select>
												</el-form-item>
												<div class="queryGroup">
													<el-input class='pairgrid-query' v-model="search_text" @keyup.enter.native="deviceQuery" :placeholder="placeholderSelect"></el-input>
											    	<i @click='deviceQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
												</div>
												<div class="editButton" size="mini" @click="addBatchSnClick('device')" v-if="templateNetType != 'egw'">
													<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
													<span><%=rb.getString("PiLiangShuRu")%></span>
												</div>
											</div>
										</template>
										<template slot='right'>
											<el-table-column v-if="templateNetType == 'enb' || templateNetType == 'gnb'" prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'enb'" prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'gnb'" prop='hostName' label='<%=rb.getString("GNBMingCheng")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'egw'" prop='serialNumber' label='<%=rb.getString("eGWBianMa")%>'></el-table-column>
											<el-table-column v-if="templateNetType == 'egw'" prop='hostName' label='<%=rb.getString("eGWMingCheng")%>'></el-table-column>
										</template>
									</el-pairgrid>
									<el-form-item prop='relDevice' style="margin: 0;" label-width="0">
										<el-input v-model='addModifyViewForm.relDevice' v-show="false"></el-input>
									</el-form-item>
								</div>
							</div>
				   		</div>
			   		</div>
		  		</div>
			</div>    		
		</div>
		<!-- KPI -->
		<div class="commonItemBox">
			<div class='commonLeftRight'>
				<div class='commonFlex'>
					<div class="itemTitle"><%=rb.getString("ZhiBiaoXuanZe")%></div>
					<span v-show='commonOperType != "view"' class="selectLimit" style='margin-left: 10px;'>(<%=rb.getString("ZuiDuoXuanZe")%>  {{kpiNumLimit}} <%=rb.getString("ZuiDuoXuanZeKPI")%>)</span>
		  		</div>
				
		  		<div class='itemBox'>
					<!--控制条件： 网元-eNB;-->
					<el-form-item v-if="templateNetType == 'enb'" label='<%=rb.getString("DengJi")%>' prop="indicatorLevel" class='basicLeftLabel'>
						<el-radio-group v-model="addModifyViewForm.indicatorLevel" @change="levelChange" :disabled="commonOperType == 'view'">
							<el-radio label="device" border>Device</el-radio>
							<el-radio label="plmn" border style='margin-left: 20px;'>PLMN</el-radio>
						</el-radio-group>
					</el-form-item>
		  			<div style='margin: 10px 0 20px ; width: 100%;' class='deviceListBox'>
						<el-ctable v-if='commonOperType == "view"' style="width:100%; height: 260px;border:1px solid #DFE2EE;" :show-pager="true" ref="viewKpis" :url="viewKpisUrl" :row-key="'kpiId'" >
							<el-table-column prop='kpiId' label='<%=rb.getString("ZhiBiaoID")%>'></el-table-column>
							<el-table-column prop='kpiName' label='<%=rb.getString("ZhiBiaoMingCheng")%>'></el-table-column>
							<el-table-column prop='isEnable' label='<%=rb.getString("CeLiangNew")%>' v-if="templateNetType == 'enb'">
                                <template slot-scope="scope">
                                    <div v-if="scope.row.isEnable == '0'">
                                        <span class='el-icon el-icon-operation-CancelMeasure commonColor'></span>
                                        <span style='margin-left: 6px;'><%=rb.getString("Fou")%></span>
                                    </div>
                                    <div v-else-if="scope.row.isEnable == '1'">
                                        <span class='el-icon el-icon-KPI-Meas commonColor'></span>
                                        <span style='margin-left: 6px;'><%=rb.getString("Shi")%></span>
                                    </div>
                                </template>
                            </el-table-column>
							<el-table-column v-if="templateNetType == 'enb'" prop='product_type' label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
						</el-ctable>
						<el-pairgrid :id="'KPISelectTable'" v-if='commonOperType != "view"'
							:rownumber="true" 
							ref="kpiTableRef" 								
							:right-url="kpiRightUrl" 
							:left-url="kpiLeftUrl" 
							:height="'260px'" 
							:row-key="'kpiId'" 
							:query-params="kpiQueryForm" 
							:title="kpiTitle"
							:limit="kpiNumLimit"
							:messages="kpiMessage" 
							@checked-change='kpisChange'>
							<template slot="left">
								<el-table-column type="selection" width="45"></el-table-column>
								<el-table-column prop='kpiId' label='<%=rb.getString("ZhiBiaoID")%>'></el-table-column>
								<el-table-column prop='kpiName' label='<%=rb.getString("ZhiBiaoMingCheng")%>'></el-table-column>
								<el-table-column v-if="templateNetType == 'enb'" prop='product_type' label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
							</template>
							<template slot='toolbar'>
								<div class='commonFlex'>
									<el-form-item v-if="templateNetType == 'enb'" label="" style=" margin: 0 0 0 20px;" class="commonFlex productTypeSelect">
										<el-cascader 
											:key="kpiCascaderKey"
											ref="kpiCascader"
											v-model="kpiCatagoryId" 
											:options="kpiFunList" 
											:props="{ 
												value: 'id', 
												label: 'text', 
												children: 'children',
												checkStrictly: true
											}"
											:show-all-levels="true"
											separator=" / "
											@change='kpiFunChange'>
										</el-cascader>
									</el-form-item>

									<el-form-item v-else label="" style=" margin: 0 0 0 20px;" class="commonFlex productTypeSelect">
										<el-select v-model="kpiCatagoryId" @change='kpiFunChange'>
											<el-option v-for="item in kpiFunList" :key="item.catagoryId" :label="item.catagoryName" :value="item.catagoryId"></el-option>
										</el-select>
									</el-form-item>

									<div class="queryGroup">
										<el-input class='pairgrid-query' v-model="kpi_search_text" @keyup.enter.native="kpiQuery"
											placeholder="<%=rb.getString("ZhiBiaoID")%> / <%=rb.getString("ZhiBiaoMingCheng")%>"></el-input>
								    	<i @click='kpiQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
									</div>
									<div class="editButton" size="mini" @click="addBatchSnClick('kpi')" v-if="templateNetType != 'egw'">
										<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
										<span><%=rb.getString("PiLiangShuRu")%></span>
									</div>
								</div>
							</template>
							<template slot='right'>
								<el-table-column prop='kpiId' label='<%=rb.getString("ZhiBiaoID")%>(<%=rb.getString("ZhiBiaoMingCheng")%>)'>
									<template slot-scope="scope">
										<span>{{scope.row.kpiId}}({{scope.row.kpiName}})</span>
						          	</template> 
								</el-table-column>
								<el-table-column prop='catagoryName' label='<%=rb.getString("ZhiBiaoGongNengJi")%>'></el-table-column>
								<el-table-column prop='isEnable' label='<%=rb.getString("CeLiangNew")%>' v-if="templateNetType == 'enb'">
                                    <template slot-scope="scope">
                                        <div v-if="scope.row.isEnable == '0'">
                                            <span class='el-icon el-icon-operation-CancelMeasure commonColor'></span>
                                            <span style='margin-left: 6px;'><%=rb.getString("Fou")%></span>
                                        </div>
                                        <div v-else-if="scope.row.isEnable == '1'">
                                            <span class='el-icon el-icon-KPI-Meas commonColor'></span>
                                            <span style='margin-left: 6px;'><%=rb.getString("Shi")%></span>
                                        </div>
                                    </template>
                                </el-table-column>
							</template>
						</el-pairgrid>
						<el-form-item prop='relKpi' style="margin: 0;" label-width="0">
							<el-input v-model='addModifyViewForm.relKpi' v-show="false"></el-input>
						</el-form-item>
					</div>
		  		</div>
			</div>    		
	   	</div>
	</el-form>
	<div class='footer' >	
		<div style='margin-left: 30px;'>
			<el-button v-show='commonOperType != "view"' type="primary" @click="submit" :disabled='saveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
			<el-button v-show='commonOperType != "view"' @click="cancel"><%=rb.getString("QuXiao")%></el-button>
			<el-checkbox v-model="is_default" true-label="true" false-label="false" :disabled = 'commonOperType == "view"'><%=rb.getString("SheWeiMoRen")%></el-checkbox>
		</div>		
	</div>

	<el-dialog class='dialogStyle' title='<%=rb.getString("TianJia")%>' width='630px' :visible.sync='batchSnDialog' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='batchSnOrKpiForm' :rules='batchSnOrKpiRules' :model='batchSnOrKpiForm' label-position="top">
			<div>
				<label>{{batchSnOrKpiLabel}}</label>
				<el-form-item prop='serialNumberOrKpi' style="margin-bottom:22px;">
					<el-input v-model='batchSnOrKpiForm.serialNumberOrKpi' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'>
					<span class='el-icon el-icon-circle-info' style='font-size:14px;'></span>
					<span style="font-size:12px;">{{batchSnOrKpiTip}}</span>
				</p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
	var kpiTemplate = new Vue({
		el: '#kpiViewTemplatePage',
		data(){
			var vm = this,
				validateTempName = function(rule,value,callback) {
					//vm.addModifyViewForm.tempName = value.replace(/\+|\ /g,'');
					if(value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingShuRuMuBanMingCheng")%>');
					}else {
						callback();
					}
				},
				validateHourWeek = function(rule,value,callback) {	
					if(vm.addModifyViewForm.reportPeriod == "1440" || vm.commonOperType == 'view'){
						//hour 复选框 1-选中 0-未选中
		            	if(vm.addModifyViewForm.checkAll == "0"){
		            		if(value === '' || value === null || value.length == 0) {
								callback('<%=rb.getString("QingXuanZeShiJian")%>');
							}else {
								callback();
							}
		            	}else{
		            		callback();
		            	}
	            	}else {
	            		callback();
	            	}
				},
				// 校验设备
				validateDevice = function(rule,value,callback) { 
					if(vm.isAllOperator == false){
						if(!vm.showDeviceGroup && value.length == 0) {
							callback('<%=rb.getString("QingXuanZeSheBei")%>');
						}else {
							callback();
						}
					}else {
						callback();
					}
				},
				//校验设备组
				validateDeviceGroup = function(rule,value,callback) { 
					if(vm.isAllOperator == false){
						if(vm.showDeviceGroup && value.length == 0) {
							callback('<%=rb.getString("QingXuanZeSheBeiZu")%>');
						}else {
							callback();
						}
					}else {
						callback();
					}
				},
				validateKpi = function(rule,value,callback) { 
					if(value.length == 0) {
						callback('<%=rb.getString("QingXuanZeZhiBiao")%>');
					}else {
						callback();
					}
				},
				validatorSnOrKpi = (rule,value,callback) => {
					var temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
						list = value.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){
							return item.length > 0;
						});
						
					if(value != null && value.length != 0 && list.length > 200){
						callback(new Error('<%=rb.getString("XianZhiYiBaiTiao200")%>'));
					}else if(value != null && value.length != 0 && list.length < 201){
						var nameFlag = list.every(function(item,index){
							return temp.test(item)
						})
						if(nameFlag){
							callback()
						}else{
							var errorTip = vm.batchInputType == 'device' ? '<%=rb.getString("QingShuRuZhengQueSn")%>' : '<%=rb.getString("QingShuRuZhengQueKPI")%>';
							callback(new Error(errorTip));
						}
						
					}else if (value == null || value.length == 0) {
						var errorTips = vm.batchInputType == 'device' ? '<%=rb.getString("SNBuNengWeiKong")%>' : '<%=rb.getString("KPIBuNengWeiKong")%>';
						callback(new Error(errorTips));
					}else{
						callback();
					}
				};
					
			return {
				is_default: 'false',
				addModifyViewForm: { 
					tempName: '',
					creator: '',
					updator: '',
					isPublic: '0',//设置为公共模板 0-私有模板   1- 公共模板
					description: '',
					timeZone: timeZone,	
					reportPeriod: '15',
					checkAll: '1',// Any: 1-所有时段； 0-指定时段
					busyTime: '',//忙时
					week: [],
					hour: [],
					selDeviceType: '2',
					indicatorLevel: 'device',
					device_type: 'ENB,GSM', //设备类型 ENB,GSM
					relKpi: '',
					relDevice: '',
					autoAccessDeviceGroupId: ''
				},
				// 校验规则
				formRules: { 
					tempName: [{validator: validateTempName}],
					hour: [{validator: validateHourWeek}],
					week: [{validator: validateHourWeek}],
					autoAccessDeviceGroupId: [{validator: validateDeviceGroup}],
					relDevice: [{validator: validateDevice}],
					relKpi: [{validator: validateKpi}],
				},
				deviceGroupForm: {
					groupId: '',
					searchText: ''
				},
				queryForm: {
					groupId: '',
					searchText: '',
					product_type: ''
				},
				kpiQueryForm: {
					catagoryId: '',
					queryType: 'mgmt',
					searchText: '',
					isEnable: '1',
					indicatorLevel: 'device', //网元等级  device-设备级别  plmn-PLMN级别
				},
				kpiCatagoryId: [],  // cascader 需要数组格式
				kpiCascaderKey: 0,  // 用于强制重新渲染 cascader
				product_type: '',
				deviceGroupSearchText: '',
				search_text:'',
				kpi_search_text: '',
				deviceNumLimit: '',
				kpiNumLimit: '',
				productTypeList: [],
				kpiFunList: [],  // 当前显示的功能集数据（可能被过滤）
				kpiFunListAll: [],  // 完整的功能集原始数据
				kpiList: [],

				groupUrl: '',
				groupRightUrl: '',

		    	leftUrl : '',
		    	rightUrl : '',

		    	kpiLeftUrl: '',
		    	kpiRightUrl: '',

		    	viewServicesUrl: '',
		    	viewServiceGroupsUrl: '',
		    	viewKpisUrl: '',
				deviceTitle: ['','<%=rb.getString("YiXuan")%>'],
				kpiTitle: ['','<%=rb.getString("YiXuan")%>'],
				defaultCheckedGroup: [],
				curProductType:'',
				deviceGroupSelection: [],
				placeholderSelect: '<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>',
				commonMessage: {
                    placeholder: '<%=rb.getString("XiaoZhanBianMa") %>'
                },
                kpiMessage: {
                    placeholder: '<%=rb.getString("ZhiBiaoID")%>'
                },
                tableCurProduct: '',
						
				commonOperType: '',
				commonRowData: [],
				comonRowDataTempId: '',
				saveBtnDisabled: false,
				
				//修改
				autoSelDeviceGroup: [],
				groupSelectedList: [],
				nameDescDisabled: false,
				isAllOperator: false,
				
				weekArr: [
					{'label': 'Sun', 'value': '7'}, //sunday 周日
					{'label': 'Mon', 'value': '1'}, //monday 周一
					{'label': 'Tue', 'value': '2'}, //Tuesday 周二
					{'label': 'Wed', 'value': '3'}, //wednesday 周三
					{'label': 'Thu', 'value': '4'}, //Thursday 周四
					{'label': 'Fri', 'value': '5'}, //friday 周五
					{'label': 'Sat', 'value': '6'}  //saturday 周六
				],
				hourArr: [
					{'label': '0:00', 'value': '0'},
					{'label': '1:00', 'value': '1'},
					{'label': '2:00', 'value': '2'},
					{'label': '3:00', 'value': '3'},
					{'label': '4:00', 'value': '4'},
					{'label': '5:00', 'value': '5'},
					{'label': '6:00', 'value': '6'},
					{'label': '7:00', 'value': '7'},
					{'label': '8:00', 'value': '8'},
					{'label': '9:00', 'value': '9'},
					{'label': '10:00', 'value': '10'},
					{'label': '11:00', 'value': '11'},
					{'label': '12:00', 'value': '12'},
					{'label': '13:00', 'value': '13'},
					{'label': '14:00', 'value': '14'},
					{'label': '15:00', 'value': '15'},
					{'label': '16:00', 'value': '16'},
					{'label': '17:00', 'value': '17'},
					{'label': '18:00', 'value': '18'},
					{'label': '19:00', 'value': '19'},
					{'label': '20:00', 'value': '20'},
					{'label': '21:00', 'value': '21'},
					{'label': '22:00', 'value': '22'},
					{'label': '23:00', 'value': '23'}
				],
				curDefaultTemplate: '',
				curIsPublic: '',
				curTemplateName: '',

				batchSnDialog: false,
				batchInputType: '', // device / kpi
				batchSnOrKpiLabel: '',
				batchSnOrKpiTip: '',
				batchSnOrKpiForm:{
					serialNumberOrKpi: '',
					type: 'input'
				},
				batchSnOrKpiRules:{
					serialNumberOrKpi:[
						{validator: validatorSnOrKpi, trigger:'change'}
					]
				},
				
			}
		},
		
		computed: {
			templateNetType() {
				return kpiQueryVue.currentNetworkType ? kpiQueryVue.currentNetworkType : sysMain.headType;
			},
			showDeviceGroup(){ 
				return this.addModifyViewForm.selDeviceType == '1';
			}
		},
		
		watch: {
			//设备 设备组选择 1： 设备组， 2- 设备
            "addModifyViewForm.busyTime":function(newVal){
				var vm = this;

				if(vm.addModifyViewForm.checkAll == '1'){
					//小时置空
					vm.addModifyViewForm.hour = [];	
					//周 需要置空
					vm.addModifyViewForm.week = [];					
				}else if(vm.addModifyViewForm.checkAll == '0'){
					// 优化：用对象映射忙时对应小时
					var busyHourMap = {
						'6': ['8','9','10','18','19','20'],
						'8': ['8','9','10','11','18','19','20','21']
					};
					vm.addModifyViewForm.hour = busyHourMap[newVal] || [];
					// 周 需要全部选中
					vm.addModifyViewForm.week = ['1','2','3','4','5','6','7'];
				}                
			},
			
            //查询粒度： 选择 24Hour（1440） 时：week，hour 不可选择； 并将指定时段默认选中为 所有时段,reportPeriod参数为'1440''；参数： week, hour 置空
            'addModifyViewForm.reportPeriod': function(val) {
                var vm = this;
                
            	if(val != '1440' && vm.commonOperType != 'view'){
            		//15min or  60min or add or view
            		vm.$refs.addModifyViewForm.validateField('week');
            		vm.$refs.addModifyViewForm.validateField('hour');
                }else{
                	//view
                	if(vm.commonOperType == 'view'){
						//查询时段： 1-所有时段， 0-指定时段
						if(vm.addModifyViewForm.week.length == 0 && vm.addModifyViewForm.hour.length == 0){
							vm.addModifyViewForm.checkAll = '1';
						}else{
							vm.addModifyViewForm.checkAll = '0';
						}
                	}else{
						// 粒度： 24小时时，忙时选中项置空
						vm.addModifyViewForm.busyTime = '';
                		vm.addModifyViewForm.week = [];
                    	vm.addModifyViewForm.hour = [];
                    	vm.$refs.addModifyViewForm.clearValidate('week');
                    	vm.$refs.addModifyViewForm.clearValidate('hour');
                		vm.addModifyViewForm.checkAll = '1';
                	}
                }
            },
            //周期设定：1-所有时段， 0-指定时段； 所有时段时，week,hour 不可选，参数： week, hour 置空
            'addModifyViewForm.checkAll': function(val) {
                var vm = this;

				//未选中all && 不是查看详情时，校验周和小时
            	if(val == '0' && vm.commonOperType != 'view'){
            		vm.$refs.addModifyViewForm.validateField('week');
            		vm.$refs.addModifyViewForm.validateField('hour');
                }else{
                	//view
                	if(vm.commonOperType == 'view'){
                		if(vm.addModifyViewForm.week.length == 0 && vm.addModifyViewForm.hour.length == 0){
							vm.addModifyViewForm.checkAll = '1';
						}else{
							vm.addModifyViewForm.checkAll = '0';
						}
                	}else{
						//选中all,忙时选中项置空
						vm.addModifyViewForm.busyTime = '';
                		vm.addModifyViewForm.week = [];
                    	vm.addModifyViewForm.hour = [];
                    	vm.$refs.addModifyViewForm.clearValidate('week');
                    	vm.$refs.addModifyViewForm.clearValidate('hour');
                	}
					
                }
            },
            //设备 设备组选择 1： 设备组， 2- 设备
            "addModifyViewForm.selDeviceType":function(newVal){
				var vm = this;
				
				if(vm.templateNetType == 'enb'){
					var deviceType;
					if(newVal == '1'){
						vm.curProductType = vm.tableCurProduct;
						//更新指标数据，增加此参数
						vm.addModifyViewForm.device_type = 'ENB,GSM';
						vm.$set(vm.kpiQueryForm, 'device_type', 'ENB,GSM');
						deviceType = 'ENB,GSM';  // 设备组模式显示全部
					}else{
						var rows = vm.$refs.cpairgrid.getData();
						vm.commonProductType(rows);
						//删除指标查询参数 device_type
						delete vm.kpiQueryForm.device_type;
						deviceType = null;  // 设备模式显示全部
					}
					
					// 根据 device_type 过滤功能集数据
					vm.filterKpiFunList(deviceType);
					
					// 清空指标模块的所有查询条件
					vm.kpi_search_text = '';  // 清空模糊查询输入框
					vm.$set(vm.kpiQueryForm, 'searchText', '');
					vm.$set(vm.kpiQueryForm, 'catagoryId', '');
					
					//清除 kpi 已选中数据
					vm.addModifyViewForm.relKpi = '';
					
					// 清空 cascader - 通过改变 key 强制重新渲染
					vm.kpiCatagoryId = [];  // cascader 需要空数组
					vm.kpiCascaderKey += 1;  // 改变 key 会导致组件完全重新创建
					
					// 使用 $nextTick 确保组件重新渲染后再操作表格
					vm.$nextTick(function(){
						if(vm.$refs.kpiTableRef){
							vm.$refs.kpiTableRef.clear();
							//vm.$refs.kpiTableRef.reload();
						}
					});
				}
			},
			//产品类型
			curProductType: function(newVal,oldVal){
				var vm = this;

				if(vm.templateNetType == 'enb'){
					if(newVal !== oldVal){
						console.log('产品类型变化：', newVal);
						//eNB 传此参数
						//vm.kpiQueryForm.product_type = newVal;
						//vm.$refs.kpiTableRef.reload();
						//现有逻辑： 当 newVal 的值包含 'RTS' 时， 指标功能集的数据需要只显示 2g 的指标功能集，
						//反之没有 ‘RTS’ 类型时，指标功能集只显示 4g 的指标功能集；
						// 当产品类型为空时，显示全部指标功能集
						var deviceType = vm.getDeviceTypeFromProductType(newVal);
						vm.setDeviceTypeForKpiQueries(deviceType);
						vm.filterKpiFunList(deviceType);
						vm.resetKpiSelection(true);
						vm.addModifyViewForm.relKpi = '';
					}
				}
			}
		},
		
		methods: {
			// 根据 device_type 过滤功能集数据，支持树形结构递归过滤
			filterKpiFunList(deviceType){
				var vm = this;
				
				if(!deviceType || deviceType == 'ENB,GSM'){
					// 显示全部数据
					vm.kpiFunList = vm.kpiFunListAll;
				} else {
					// 递归过滤树形数据
					var allowedTypes = deviceType.split(',').map(function(t){ return t.trim(); });
					vm.kpiFunList = vm.kpiFunListAll.filter(function(item){
						return vm.shouldIncludeNode(item, allowedTypes);
					}).map(function(item){
						return vm.filterNodeChildren(item, allowedTypes);
					});
				}
				
				console.log('过滤功能集 - device_type:', deviceType, '结果数量:', vm.kpiFunList.length);
			},
			// 判断节点是否应该包含，支持多个类型
			shouldIncludeNode(node, allowedTypes){
				if(!node) return false;
				
				// ALL 节点永远包含
				if(node.id === '' || node.text === 'ALL'){
					return true;
				}
				
				// 检查当前节点是否匹配
				if(allowedTypes.indexOf(node.device_type) >= 0){
					return true;
				}
				
				// 检查子节点是否有匹配
				if(node.children && Array.isArray(node.children)){
					return node.children.some(function(child){
						return allowedTypes.indexOf(child.device_type) >= 0;
					});
				}
				
				return false;
			},
			// 递归过滤节点的子节点
			filterNodeChildren(node, allowedTypes){
				var vm = this;
				var filtered = Object.assign({}, node);
				
				if(filtered.children && Array.isArray(filtered.children)){
					filtered.children = filtered.children.filter(function(child){
						return allowedTypes.indexOf(child.device_type) >= 0 || child.device_type === '' || child.id === '';
					});
					
					// 递归处理子节点的子节点
					filtered.children = filtered.children.map(function(child){
						return vm.filterNodeChildren(child, allowedTypes);
					});
				}
				
				return filtered;
			},
			getDeviceTypeFromProductType(productType){
				if(!productType){
					return null;
				}
				var upper = (productType || '').toString().toUpperCase();
				var tokens = upper.split(',').map(function(item){
					return item && item.trim();
				}).filter(function(item){
					return item && item !== 'ALL';  // 过滤空和ALL
				});
				
				if(tokens.length == 0){
					return null;
				}
				
				// 判断包含哪些类型的产品
				var hasRTS = tokens.indexOf('RTS') >= 0;  // GSM类产品
				var hasOther = tokens.some(function(item){
					// QRTB/BAIBLQ/QAFA/BLX/BSC/MLQ/MLN 都是 ENB 类产品
					return item !== 'RTS';
				});
				
				// 根据包含的产品类型返回应该显示的功能集
				if(hasRTS && hasOther){
					// 既有GSM又有ENB的混合产品，显示两种
					return 'ENB,GSM';
				} else if(hasRTS){
					// 只有 RTS (GSM 类产品)
					return 'GSM';
				} else {
					// 其他产品类型 (ENB 类产品)
					return 'ENB';
				}
			},
			setDeviceTypeForKpiQueries(deviceType){
				var vm = this;
				if(deviceType){
					vm.$set(vm.kpiQueryForm, 'device_type', deviceType);
				}else if(vm.kpiQueryForm && vm.kpiQueryForm.hasOwnProperty('device_type')){
					delete vm.kpiQueryForm.device_type;
				}
			},
			resetKpiSelection(clearTable){
				var vm = this;
				vm.kpiCatagoryId = [];
				vm.kpiCascaderKey += 1;
				vm.kpi_search_text = '';
				if(vm.kpiQueryForm){
					vm.kpiQueryForm.searchText = '';
					vm.kpiQueryForm.catagoryId = '';
				}
				if(clearTable){
					vm.$nextTick(function(){
						if(vm.$refs.kpiTableRef){
							vm.$refs.kpiTableRef.clear();
						}
					});
				}
			},
			// 统一处理指标功能集的过滤更新流程
			updateKpiFiltering(deviceType){
				var vm = this;
				
				// 过滤功能集数据
				vm.filterKpiFunList(deviceType);
				
				// 清空所有查询条件
				vm.kpi_search_text = '';
				if(vm.kpiQueryForm){
					vm.kpiQueryForm.searchText = '';
					vm.kpiQueryForm.catagoryId = '';
				}
				
				// 清除已选指标数据
				vm.addModifyViewForm.relKpi = '';
				
				// 重置级联选择器和表格
				vm.kpiCatagoryId = [];
				vm.kpiCascaderKey += 1;
				
				vm.$nextTick(function(){
					if(vm.$refs.kpiTableRef){
						vm.$refs.kpiTableRef.clear();
					}
				});
			},
			// rowData： 修改，查看时模板行数据； networkCode： 当前操作网元； operType：操作类型-新建，修改，查看模板
			//20230106 将涉及的网元条件去除，使用全局网元
			init(rowData,operType){
				var vm = this, 
					curGroupListUrl = '', 
					params = {
						isShowAll: '1',
						noKpiShowGroup: '0'
					};

				vm.commonOperType = operType;
				
				closeLoading();

				//设备组列表
				if(operType == 'add'){
					vm.groupUrl = '${ctx}/pm/template/getTempDeviceGroupList.action';
				}else{
					vm.commonRowData = rowData;
					vm.comonRowDataTempId = rowData.tempId;// 获取当前模板的id
					vm.queryForm.tempId = rowData.tempId;
					vm.groupUrl = '${ctx}/pm/template/getTempDeviceGroupList.action?tempId='+rowData.tempId;
					
					vm.getKPITemlateInfo(rowData.tempId);
					
					//内置的Basic模板不允许修改名称和描述; 复制模板时可以修改名称和描述
					if(rowData.isCustomize == '0' && operType != 'copy'){
						vm.nameDescDisabled = true;
					}
				}

				if(vm.templateNetType == 'enb'){
					//获取产品类型
					vm.getProductList();

					vm.placeholderSelect = '<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>';
					vm.commonMessage =  {
		                placeholder: '<%=rb.getString("XiaoZhanBianMa") %>'
	                };
					//curGroupListUrl = '${ctx}/pm/indicatormg/getIndicatorGroupList.action';
					curGroupListUrl = '${ctx}/pm/indicatormg/getIndicatorGroupTree.action';
					if(vm.commonOperType == 'add'){
						vm.leftUrl = '${ctx}/pm/template/getEnbListPageData.action?isGnb=0';
						//isUseTemplate 参数区分 getEnbListPageData接口 在新建模板查询设备与自定义图表查询设备是否处理 plmnId 和 bts_id
						vm.queryForm.isUseTemplate = 'true'; 
						vm.kpiLeftUrl = '${ctx}/pm/indicatormg/getIndicatorListByPage.action';
					}else if(vm.commonOperType == 'modify' || vm.commonOperType == 'copy'){
						vm.leftUrl = '${ctx}/pm/template/getModifyEnbListPageData.action';						
						vm.kpiLeftUrl = '${ctx}/pm/indicatormg/getIndicatorListByPage.action';
						vm.kpiRightUrl = '${ctx}/pm/template/getSelectedIndicatorListPageData.action?tempId='+vm.comonRowDataTempId;
						vm.kpiQueryForm.product_type = vm.curProductType; // kpi默认的产品类型
						
					}else{
						vm.viewServicesUrl = '${ctx}/pm/template/viewTemplateRelDeviceList.action?tempId='+vm.comonRowDataTempId;
						vm.viewServiceGroupsUrl = '${ctx}/pm/template/getSelDeviceGroupList.action?tempId='+vm.comonRowDataTempId;
						vm.viewKpisUrl = '${ctx}/pm/template/viewTemplateRelIndicatorsList.action?tempId='+vm.comonRowDataTempId;
					}
					
				}else if( vm.templateNetType == 'gnb'){
					vm.placeholderSelect = '<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("GNBMingCheng")%>';
					vm.commonMessage =  {
		                placeholder: '<%=rb.getString("XiaoZhanBianMa") %>'
	                };
					curGroupListUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupList.action';	

					if(vm.commonOperType == 'add'){
						vm.leftUrl = '${ctx}/pm/template/getEnbListPageData.action?isGnb=1'		
                        vm.queryForm.isUseTemplate = 'true'; 				
						vm.kpiLeftUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData.action';
					}else if(vm.commonOperType == 'modify'  || vm.commonOperType == 'copy'){
						vm.leftUrl = '${ctx}/gnb/pm/template/getModifyEnbListPageData.action';						
						vm.kpiLeftUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData.action';
						vm.kpiRightUrl = '${ctx}/gnb/pm/template/getSelectedIndicatorListPageData.action?tempId='+vm.comonRowDataTempId;
					}else{
						vm.viewServicesUrl = '${ctx}/gnb/pm/template/viewTemplateRelDeviceList.action?tempId='+vm.comonRowDataTempId;
						vm.viewServiceGroupsUrl = '${ctx}/gnb/pm/template/getSelDeviceGroupList.action?tempId='+vm.comonRowDataTempId;
						vm.viewKpisUrl = '${ctx}/gnb/pm/template/viewTemplateRelIndicatorsList.action?tempId='+vm.comonRowDataTempId;
					}
					
				}else{  
					vm.placeholderSelect = '<%=rb.getString("eGWBianMa")%> / <%=rb.getString("eGWMingCheng")%>';
					vm.commonMessage =  {
		                placeholder: '<%=rb.getString("eGWBianMa") %>'
	                };
					curGroupListUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupList.action';

					if(vm.commonOperType == 'add'){
						vm.leftUrl = '${ctx}/egw/pm/template/getEgwListPageData.action'; 						
						vm.kpiLeftUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListPageData.action';
					}else if(vm.commonOperType == 'modify' || vm.commonOperType == 'copy'){
						vm.leftUrl = '${ctx}/egw/pm/template/getModifyEgwListPageData.action';						
						vm.kpiLeftUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListPageData.action';
						vm.kpiRightUrl = '${ctx}/egw/pm/template/getSelectedIndicatorListPageData.action?tempId='+vm.comonRowDataTempId;
						
					}else{
						vm.viewServicesUrl = '${ctx}/egw/pm/template/viewTemplateRelDeviceList.action?tempId='+vm.comonRowDataTempId;
						vm.viewServiceGroupsUrl = '${ctx}/egw/pm/template/getSelDeviceGroupList.action?tempId='+vm.comonRowDataTempId;
						vm.viewKpisUrl = '${ctx}/egw/pm/template/viewTemplateRelIndicatorsList.action?tempId='+vm.comonRowDataTempId;
					}
				}

				//-----------------------add
				//获取指标功能集
				axios.post(curGroupListUrl,stringify(params)).then(function(response){
					var data = response.data;

					if(vm.templateNetType == 'enb'){
					
						var rowData = data || [];
						// 递归删除空的 children 数组
						function removeEmptyChildren(data) {
							if (Array.isArray(data)) {
								data.forEach(function(item) {
									removeEmptyChildren(item);
								});
							} else if (data && typeof data === 'object') {
								if (Array.isArray(data.children)) {
									if (data.children.length === 0) {
										delete data.children;
									} else {
										removeEmptyChildren(data.children);
									}
								}
							}
						}
						
						removeEmptyChildren(rowData);
						
						// 在数据开头添加"全部"选项
						rowData.unshift({
							"id": "",
							"text": "ALL",
							"device_type": "",
							"iconCls": "null"
						});
						
						// 保存完整数据
						vm.kpiFunListAll = rowData;
						// 初始显示全部数据
						vm.kpiFunList = rowData;
					}else{
						vm.kpiFunList = data || [];
					}					
				}).catch(function(error){})
				
				//kpi, device 选择数量限制 4G,5G,eGW 此接口不做区分
			    $.post("${ctx}/cell/perfmgmt/kpitemp/getLimitInfo.action",{}, function(data) {
					if(data){
						vm.deviceNumLimit = parseInt(data.deviceNum);
						vm.kpiNumLimit = parseInt(data.indicatorNum);
					}		 							
				}, "json");
			},
			//-------------------------------------------- 修改详情
			//获取已选设备的数据对应的产品类型
			getSelectRowsProduct(){
				var vm = this;

				//获取已选设备对应的产品类型，来动态更新指标数据
				axios.post('${ctx}/pm/template/getSelectedEnbListPageData.action?tempId='+vm.comonRowDataTempId).then(function(response){
					var data = response.data;

					if(data.rows){
						vm.commonProductType(data.rows);
					}				
				}).catch(function(error){})	
			},
			//获取模板详情
			getKPITemlateInfo(tempId){
				var vm = this, 
					curModifyTemplateInfoUrl = '',
					params = {
						tempId: tempId,
						timeZone: timeZone
					};

				if(vm.commonRowData.creator == '' || vm.commonRowData.creator == null || vm.commonRowData.creator == undefined){
					params.creator = vm.commonRowData.updator;
				}else{
					params.creator = vm.commonRowData.creator;
				}
				if( vm.templateNetType == 'enb'){
			    	curModifyTemplateInfoUrl = '${ctx}/pm/template/getTemplateInfo.action';	    	
			    }else if( vm.templateNetType == 'gnb'){
			    	curModifyTemplateInfoUrl = '${ctx}/gnb/pm/template/getTemplateInfo.action';
			    }else{
			    	curModifyTemplateInfoUrl = '${ctx}/egw/pm/template/getTemplateInfo.action';	
			    }
				
				axios.post(curModifyTemplateInfoUrl, stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						if(vm.commonOperType == 'copy'){
							vm.addModifyViewForm.tempName = data.tempName + '_copy';
						}else{
							vm.addModifyViewForm.tempName = data.tempName;
						}
						vm.curTemplateName = data.tempName;
						vm.addModifyViewForm.isPublic = data.isPublic;
						vm.curIsPublic = data.isPublic;
						vm.addModifyViewForm.creator = data.creator;
						vm.addModifyViewForm.updator = data.updator;
						vm.addModifyViewForm.description = data.description;
						vm.addModifyViewForm.reportPeriod = data.reportPeriod;
						//粒度： 24Hour
						if(data.reportPeriod == '1440'){
							
						}else{
							var curWeek = data.week;
							if(curWeek == '' || curWeek == null || curWeek == undefined){
								vm.addModifyViewForm.week = [];
							}else{
								var endWeek = curWeek.split('0,').join('');
								vm.addModifyViewForm.week = endWeek.split(',');
							}
							if(data.hour == '' || data.hour == null || data.hour == undefined){
								vm.addModifyViewForm.hour = [];
							}else{
								vm.addModifyViewForm.hour = data.hour.split(','); 
							}
							//查询时段： 1-所有时段， 0-指定时段, 周和小时都没选时，小时应被选中
							if(vm.addModifyViewForm.week.length != 0 && data.week != '' && vm.addModifyViewForm.hour.length != 0 && data.hour != ''){
								vm.addModifyViewForm.checkAll = '0';
							}else{
								vm.addModifyViewForm.checkAll = '1';
							}
						}
						//设备 设备组选择 1： 设备组， 2- 设备
						vm.addModifyViewForm.selDeviceType = data.selDeviceType;
						if(data.selDeviceType == '1'){
							var deviceGroupId = vm.templateNetType == 'enb' ? data.autoAccessDeviceGroup: data.auto_access_device_group;
							var dckList = deviceGroupId ? deviceGroupId.split(',') : [];
							vm.curProductType = vm.tableCurProduct;
							vm.defaultCheckedGroup = dckList.map((item)=>{ return item * 1; });//转换为数字
						}else{
							vm.getSelectRowsProduct();
						}
						 // 往设备选择的右表静态载入数据并初始化级联

						if(data.relDevice === '' || data.relDevice === null || data.relDevice === undefined){
							vm.addModifyViewForm.relDevice = '';
						}else{
							vm.addModifyViewForm.relDevice = data.relDevice;	

							if(vm.commonOperType == 'modify' || vm.commonOperType == 'copy'){
								if( vm.templateNetType == 'enb'){
									vm.rightUrl = '${ctx}/pm/template/getSelectedEnbListPageData.action?tempId='+vm.comonRowDataTempId;
								}else if( vm.templateNetType == 'gnb'){
									vm.rightUrl = '${ctx}/gnb/pm/template/getSelectedEnbListPageData.action?tempId='+vm.comonRowDataTempId;
								}else{
									vm.rightUrl = '${ctx}/egw/pm/template/getSelectedEgwListPageData.action?tempId='+vm.comonRowDataTempId;
								}
							}
						}
						
						if (vm.templateNetType == 'enb') {
							vm.addModifyViewForm.indicatorLevel = data.indicatorLevel ? data.indicatorLevel : 'device';
							vm.kpiQueryForm.indicatorLevel = data.indicatorLevel ? data.indicatorLevel : 'device';

							//针对与 设备组
							if(vm.addModifyViewForm.selDeviceType == '1'){
								vm.addModifyViewForm.device_type = data.device_type ? data.device_type : 'ENB,GSM';
								vm.kpiQueryForm.device_type = data.device_type ? data.device_type : 'ENB,GSM';
							}
						}
						if(data.relKpi === '' || data.relKpi === null || data.relKpi === undefined){
							vm.addModifyViewForm.relKpi = '';
						}else{
							 vm.addModifyViewForm.relKpi = data.relKpi;	
						}
						
						//不显示 设备，设备组模块；
						if(data.isAllOperator == 1 && vm.commonOperType != 'copy'){
							vm.isAllOperator = true;
							//此模板不显示设备，需将已勾选设备的产品类型置空
							vm.curProductType = '';
						}
						//是否为默认模板标识
						vm.is_default = data.is_default;
						vm.curDefaultTemplate = data.is_default;
					}
				})
	
				setTimeout(function(){
					initForm(vm.$refs.addModifyViewForm);
				},600)
			},
			
			
			//-------------------------------------------- select devices or select device Groups
			//获取产品类型
			getProductList(){
				var vm = this;
				
				axios.post('${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0&isAll=1').then(function(response){
					var data = response.data;

					// 动态删除 BTS
					data = data.filter(function(item) {
						return item !== 'BTS';
					});
					if(data.length == 0){
						vm.queryForm.product_type = 'no';
					}else{
						//全部的产品类型
						var curProduct =  'ALL,' + data.join(',');
						vm.tableCurProduct = curProduct;
						vm.curProductType = curProduct;
						//eNB 传此参数
						if( vm.templateNetType == 'enb' && vm.commonOperType == 'add'){
							//vm.kpiQueryForm.product_type = data.join(','); // kpi默认的产品类型
						}else{
							//vm.kpiQueryForm.product_type = '';
						}
						
						//搜索下拉框数据映射
						var arr = [{label:'<%=rb.getString("QuanBu")%>',value:''}];
						data.map(function(item){
							if (item){								
								arr.push({label:item,value:item})
							}
						});
						vm.productTypeList = arr;
					}
				}).catch(function(error){})
			},
			//device 产品类型改变
			productChange(val){
				var vm = this;

				vm.queryForm.product_type = val;			
			},
			//kpi 指标功能集改变
			kpiFunChange(val){
				var vm = this;
				
				// 确保 catagoryId 是单个值，不是数组
				// 如果 val 是数组，取最后一个元素（叶子节点的值）
				if(Array.isArray(val)){
					vm.kpiQueryForm.catagoryId = val[val.length - 1];
				} else {
					vm.kpiQueryForm.catagoryId = val;
				}
				
				console.log('功能集选中值:', val, '实际使用值:', vm.kpiQueryForm.catagoryId);
			},
			// 设备组的查询条件点击，更新 kpi 数据
			deviceTypeChange(val){
				var vm = this;
				
				// 先清空所有查询条件
				vm.kpiCatagoryId = [];  // cascader 需要空数组
				vm.kpi_search_text = '';  // 清空模糊查询输入框
				if(vm.kpiQueryForm) {
					vm.kpiQueryForm.searchText = '';
					vm.kpiQueryForm.catagoryId = '';  // 清空查询参数中的功能集ID
				}
				
				// 根据 device_type 过滤功能集数据
				vm.filterKpiFunList(val);
				
				// 清空并重新渲染 cascader
				vm.kpiCascaderKey += 1;
				
				//更新指标数据，增加此参数
				vm.kpiQueryForm.device_type = val;
				
				// 清空已选指标数据
				vm.addModifyViewForm.relKpi = '';
				
				// 重新加载表格数据
				vm.$nextTick(function(){
					if(vm.$refs.kpiTableRef){
						vm.$refs.kpiTableRef.clear();
						//vm.$refs.kpiTableRef.reload();
					}
				});
			},
			// kpi 等级查询
			levelChange(val){
				var vm = this;

				vm.kpiQueryForm.indicatorLevel = val;
				// 清除KPI的模糊查询输入框和参数
				vm.kpi_search_text = '';
				if(vm.kpiQueryForm) vm.kpiQueryForm.searchText = '';

				// 等级变化时，清空已选指标数据
				vm.addModifyViewForm.relKpi = '';
				
				// 清空 pairgrid 表格的左侧选中项和右侧已选数据
				vm.$nextTick(function(){
					if(vm.$refs.kpiTableRef){
						vm.$refs.kpiTableRef.clear();;
					}
				});
			},
			deviceGroupQuery(){
				var vm = this;

				vm.deviceGroupForm.searchText = vm.deviceGroupSearchText;
			},
			//devices 表格搜索
			deviceQuery(){
				var vm = this;

				vm.queryForm.searchText = vm.search_text;
			},
			kpiQuery(){
				var vm = this;

				vm.kpiQueryForm.searchText = vm.kpi_search_text;
			},
			//设备组选择改变
			queryGroupChange(row){ 
				var vm = this;

				if(row) {
					vm.queryForm.groupId = row.id;
				}
			},
			// 设备组列表-当选择项发生变化时触发此事件
			groupChange(selection) {
				var vm = this;
				
				vm.$nextTick(function(){
					if(vm.templateNetType == 'enb'){
						var rows = vm.$refs.groupTable.getData();
						vm.addModifyViewForm.autoAccessDeviceGroupId = rows.map(function(row){
							return row.id;
						}).join(',');

					}else{
						vm.deviceGroupSelection = selection;
						setTimeout(function(){
							var cks = vm.$refs.groupTable.getChecked();
							vm.addModifyViewForm.autoAccessDeviceGroupId = cks.join(',');
						},0)
					}
				})
			},
			// 设备选择变化时，更新选择设备记录
			devicesChange(selection) {
				var vm = this;
				
				vm.$nextTick(function(){
					var rows = vm.$refs.cpairgrid.getData();
					vm.addModifyViewForm.relDevice = rows.map(function(row){ 
						return row.smallCellCode;
					}).join(',');

					if(vm.templateNetType == 'enb'){
						vm.commonProductType(rows);
					}
				})		
			},
			// 设备选择变化时，更新选择设备记录
			kpisChange(selection) {
				var vm = this;
				
				vm.$nextTick(function(){
					var rows = vm.$refs.kpiTableRef.getData();
					vm.addModifyViewForm.relKpi = rows.map(function(row){ 
						return row.kpiId;
					}).join(',');
				})		
			},
			commonProductType(rows){
				var vm = this, 
					curProductArr = [], 
					newProductArr = [];
				
				(rows || []).map((item)=>{
					if(item.hasOwnProperty('product') && item.product != null &&　item.product != undefined && item.product != ''){
						curProductArr.push(item.product)	
					}						
				}); 
				if(curProductArr.length > 0){
					//产品类型去重
					curProductArr.map(function(item){
						if(newProductArr.indexOf(item) == -1){
							newProductArr.push(item)
						}
					});
					vm.curProductType = newProductArr.join(',');
				}else{
					vm.curProductType = vm.tableCurProduct;
				}
			},
			
			//批量添加
			addBatchSnClick(type){
				var vm = this;

				vm.batchInputType = type;
				vm.batchSnOrKpiLabel = type == 'device' ? '<%=rb.getString("XiaoZhanBianMa")%>' : '<%=rb.getString("ZhiBiaoID")%>';
				vm.batchSnOrKpiTip = type == 'device' ? '<%=rb.getString("eNBZhuCeTiShiWenZi")%>' : '<%=rb.getString("PiLiangZhuCeTiShiWenZi")%>';
				vm.batchSnDialog = true;
			},
			saveBatchSn(){
				var vm = this, 
					saveBatchUrl = '', 
					params = {}, 
					snStr = vm.batchSnOrKpiForm.serialNumberOrKpi,
					list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
						return item.length > 0;
					});
													
				//当前仅支持 eNB,gNB
				if(vm.batchInputType == 'device'){
					params.serialNumber = list.join(",");

					params.isGnb = vm.templateNetType == 'enb' ? 0 : 1;
					saveBatchUrl = '${ctx}/pm/template/getEffectiveDevices.action';
				}else{
					//批量输入 KPI ID, 支持 eNB,gNB
					if(vm.templateNetType == 'enb'){
						saveBatchUrl = '${ctx}/pm/indicatormg/getEffectiveIndicators.action';
						params.product_type = vm.kpiQueryForm.product_type;
						params.indicator_level = vm.kpiQueryForm.indicatorLevel;
					}else{
						saveBatchUrl = '${ctx}/gnb/pm/indicatormg/getEffectiveIndicators.action';
					}
					
					params.indicator_ids = list.join(",");
				}
				vm.$refs.batchSnOrKpiForm.validate((valid) => {
					if(valid){
						axios.post(saveBatchUrl, stringify(params)).then((res)=>{
							var data = res.data;
							
							if(data && data.length > 0){	
								if(vm.batchInputType == 'device'){	
									vm.$refs.cpairgrid.appendCheckedRows(data);
								}else{
									vm.$refs.kpiTableRef.appendCheckedRows(data);
								}
								vm.closeBatchSn();
							}else{
								var tip = vm.batchInputType == 'device' ? '<%=rb.getString("MeiYouKePiPeiSheBei")%>' : '<%=rb.getString("MeiYouKePiPeiZhiBiaoID")%>';
								vm.$message(tip)
							}
						})
					}
				})
			},
			closeBatchSn(){
				var vm = this;

				vm.batchSnDialog = false;
				vm.$refs.batchSnOrKpiForm.resetFields();
			},

			//保存
			submit(){
				var vm = this, 
					curSaveUrl = '', 	
					params = {}, 
					defaultChanged = false, 
					isPublicChanged = false, 
					templateNameChanged = false;
				
				if(vm.templateNetType == 'enb'){
					curSaveUrl = '${ctx}/pm/template/addOrModifyTemplate.action';
				}else if(vm.templateNetType == 'gnb'){
					curSaveUrl = '${ctx}/gnb/pm/template/addOrModifyTemplate.action';
				}else{
					curSaveUrl = '${ctx}/egw/pm/template/addOrModifyTemplate.action';
				}
				vm.$refs.addModifyViewForm.validate(function(valid){
					if(valid){
						
						params.tempName = vm.addModifyViewForm.tempName;
						params.isPublic = vm.addModifyViewForm.isPublic; //是否为公共模板
						
						params.description = vm.addModifyViewForm.description;
						params.reportPeriod = vm.addModifyViewForm.reportPeriod;
						params.week = vm.addModifyViewForm.week.join(',');
						params.hour = vm.addModifyViewForm.hour.join(',');
						params.relKpi = vm.addModifyViewForm.relKpi;
						params.timeZone = timeZone;
						params.is_default = vm.is_default;
						params.selDeviceType = vm.addModifyViewForm.selDeviceType;
						//等级受网元控制
						if(vm.templateNetType == 'enb'){
							params.indicatorLevel = vm.addModifyViewForm.indicatorLevel;

							//针对于设备组
							if(vm.addModifyViewForm.selDeviceType == '1'){
								params.device_type = vm.addModifyViewForm.device_type;
							}
						}

						// 是否为copy 模板
						params.type = vm.commonOperType == 'copy' ? 'copy' : '';

						//1-设备，2-设备组
						//区分新建，修改涉及的 设备，设备组参数
						if(vm.commonOperType == 'modify'){
							if(vm.commonRowData.creator == '' || vm.commonRowData.creator == null || vm.commonRowData.creator == undefined){
								params.creator = vm.commonRowData.updator;
							}else{
								params.creator = vm.commonRowData.creator;
							}
						}
						if(vm.commonOperType == 'add'){
							if(vm.addModifyViewForm.selDeviceType == '1'){
								params.autoAccessDeviceGroupId = vm.addModifyViewForm.autoAccessDeviceGroupId;
							}else if(vm.addModifyViewForm.selDeviceType == '2'){
								params.relDevice = vm.addModifyViewForm.relDevice;
							}
						}else if(vm.commonOperType == 'modify' || vm.commonOperType == 'copy'){
							//isAllOperator: 1时，不显示设备，设备组模块，此时vm.isAllOperator = true;
							if(vm.isAllOperator == false){
								if(vm.addModifyViewForm.selDeviceType == '1'){
									params.autoAccessDeviceGroupId = vm.addModifyViewForm.autoAccessDeviceGroupId;
								}else if(vm.addModifyViewForm.selDeviceType == '2'){
									params.relDevice = vm.addModifyViewForm.relDevice;
								}
							}
							params.tempId = vm.comonRowDataTempId;
						}
						
						var curSnLength = vm.addModifyViewForm.relDevice.split(',').length,
							curkpiLength = vm.addModifyViewForm.relKpi.split(',').length;
						//必须为当前选中的是设备列表，并不是设备组
						if((curSnLength > Number(vm.deviceNumLimit) || curkpiLength > Number(vm.kpiNumLimit)) && vm.addModifyViewForm.selDeviceType == '2'){
							vm.$message('<%=rb.getString("ZuiDuoXuanZe")%>' + vm.deviceNumLimit + '<%=rb.getString("ZuiDuoXuanZeDevice")%>, '+vm.kpiNumLimit+'<%=rb.getString("ZuiDuoXuanZeKPI")%>');
							return
						}else {
							vm.saveBtnDisabled = true;
						}
						if(vm.commonOperType == 'modify' || vm.commonOperType == 'copy'){
							if(vm.is_default == vm.curDefaultTemplate){
								defaultChanged = false;
							}else{
								defaultChanged = true;
							}
							//复选框无变化，直接提交
							if(vm.addModifyViewForm.isPublic == vm.curIsPublic ){
								isPublicChanged = false;
							}else{
								isPublicChanged = true;
							}
							//复选框无变化，直接提交
							if(vm.addModifyViewForm.tempName == vm.curTemplateName ){
								templateNameChanged = false;
							}else{
								templateNameChanged = true;
							}
							
							//复选框有变化可直接提交
							if(defaultChanged == true || isPublicChanged == true || templateNameChanged == true){
								vm.commonSubmit(curSaveUrl, params);
							}else{
								// 复选框无变化，需校验表单是否有变化，方可提交
								if(isFormChanged(vm.$refs.addModifyViewForm)) {
									vm.commonSubmit(curSaveUrl, params);
								}else{
									vm.$message('<%=rb.getString("WuCanShuBianHua")%>');
									vm.saveBtnDisabled = false;
								}
							}
						}else{								
							vm.commonSubmit(curSaveUrl, params);
						}
					}
				}) 
			},
			commonSubmit(curSaveUrl,params){
				var vm = this;
				
				axios.post(curSaveUrl, stringify(params)).then(function(response){
                	var data = response.data;
                	if(data){
                		if(data['success'] == true) {
                            vm.$message({
	    						message: '<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					});
                            //左侧模板列表更新
                            kpiQueryVue.queryTemplateList();
                            kpiQueryVue.$refs.templateSlider.hide();
                            
                            if(vm.commonOperType == 'modify'){
								//修改时，关闭的是当前修改模板对应的普通图表和自定义图表页面
								$("#winChartCondition_" + vm.comonRowDataTempId).hide(); 
								$("#winCustomChartsCondition_" + vm.comonRowDataTempId).hide();
								
								kpiQueryVue.closeCustomChartsWin(this, vm.comonRowDataTempId);
							}
                        }else {
							//如果 data 中含有 flag 字段，则说明已选指标中含有与等级不匹配的数据；
							if(vm.templateNetType == 'enb' && data.hasOwnProperty('flag') && data['flag'] == 'Indicator level mismatch'){
								vm.$message.error("<%=rb.getString("ZhiBiaoXuanZeShuJuYuDengJiBUFu")%>");
							}else{
                            	vm.$message.error(data['message']);
							}
                        }
                		vm.saveBtnDisabled = false;
                	}
                }).catch(function(){});
			},
			// 关闭新建弹窗
			cancel(){
				var vm = this;

				if(isFormChanged(vm.$refs.addModifyViewForm)){
					vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>', '<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						kpiQueryVue.$refs.templateSlider.hide();
					}).catch(() => {
						
					})
				}else{
					kpiQueryVue.$refs.templateSlider.hide();
				}
			},
		},
		
		mounted(){
			// 页面初始化时，非enb网元删除kpiQueryForm.indicatorLevel，防止5G等带空字段
			if (this.templateNetType !== 'enb') {
				this.kpiQueryForm && delete this.kpiQueryForm.indicatorLevel;
				this.kpiQueryForm && delete this.kpiQueryForm.device_type;
			}
			eventBus.$off('action-templateInit').$on('action-templateInit',this.init);
			eventBus.$off('action-submit').$on('action-submit',this.submit);
			eventBus.$off('action-cancel').$on('action-cancel',this.cancel);
		}
	});
</script>