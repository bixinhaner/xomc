<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	#backupRestoreTask .el-form-item__label{
		line-height:26px;
		width:160px;
		text-align:left;
	}
	#backupRestoreTask .el-select .el-input.is-disabled .el-input__inner{
		min-height:26px;
		max-height:26px;
	}
	#backupRestoreTask .newBackupTaskWarp .basicInfoWarp,
	#backupRestoreTask .newBackupTaskWarp .selectDeviceWarp{
		border-bottom:1px solid #E9E9E9;
	}
	#backupRestoreTask .newBackupTaskWarp .basicInfoWarp .basicInfo{
		padding:40px 50px 20px;
	}
	#backupRestoreTask .newBackupTaskWarp .selectDeviceWarp .selectDeviceInfo{
		padding:30px 50px 20px;
	}
	#backupRestoreTask .newBackupTaskWarp .newTaskName{
		margin-left:76px;
	}
	#backupRestoreTask .newBackupTaskWarp .basicInfoWarp .newTaskName .el-input{
		width:348px;
		height:26px;
		line-height:26px;
	}
	#backupRestoreTask .newBackupTaskWarp .basicInfoWarp .newTaskName .el-form-item__error{
		margin-left:160px;
	}
	#backupRestoreTask .selectDeviceBox{
		margin: 0 72px;
		background:#FFFFFF;
		display:flex;
		height:300px;
	}
	#backupRestoreTask .selectDeviceBox .el-radio{
		margin-right:30px;	
	}
	#backupRestoreTask .leftDeviceSpecific {
		width:260px;
		height:298px;
		border:1px solid #E9E9E9;
	}
	#backupRestoreTask .leftDeviceSpecific p{
		height:36px;
		line-height:36px;
		text-align:center;
		background:#F6F7FB;	
		border-bottom:1px solid #E9E9E9;
	}
	#backupRestoreTask .leftDeviceSpecific .el-radio__label{
		font-size:12px;
	}
	#backupRestoreTask .executeModeBox{
		padding: 20px 0 0 76px;
		display:flex;
	}

	#backupRestoreTask .timeSelect{
		display:inline-block;
		vertical-align:bottom;
		margin-left:15px;
		margin-bottom:25px;
		margin-top:-4px;
	}
	#backupRestoreTask .closeSlide{
		position:absolute;
		right:30px;
		top:20px;
	}
	#backupRestoreTask .el-pairgrid .el-pairgrid-title{
		display: none;
	}
	#backupRestoreTask .el-pairgrid .el-pairgrid-title:last-child{
        top: 10px;
        right: 10px !important;
	}
	#backupRestoreTask .pairgrid-left .el-query{
		margin-left:0 !important;
	}
	#backupRestoreTask .pairgrid-right{
		top:38px !important;
		border:1px solid #E9E9E9;
	}
	#backupRestoreTask .noSelectDevice .el-form-item__error{
		/*left:50px;*/
		margin-left: 76px;
	}
	#backupRestoreTask .el-radio{
		font-weight:unset;
	}
	#backupRestoreTask .el-icon-time{
		font-size:14px;
	}
	#backupRestoreTask .searchCon{
		margin-left:18px;
	} 
	#backupRestoreTask .searchCon .el-input--suffix{
		width:460px; 
		height:28px;
	}
	#backupRestoreTask .searchCon .el-input--suffix .el-input__inner{
		height:28px;
		line-height:28px;
		padding-left:15px;
		padding-right:40px;
		margin-top: 1px;
	}
	#backupRestoreTask .searchCon .el-icon-common-search{
		margin-top:5px;
		margin-right:10px;
	}
	#backupRestoreTask .boxBorderCon .el-radio-group{
		margin-top:6px;
	}
	#backupRestoreTask .infoTip{
		font-size:14px;
		margin-right:5px;
	}
	#backupRestoreTask .infoTip:before{
		color:#4D84FF;
	}
	#backupRestoreTask .el-radio.is-bordered { 
		max-width: 160px; 
		height: 30px; 
		padding: 7px 12px; 
	}
	#backupRestoreTask .editButton{
		position: absolute;
		right: 130px;
		top: -5px;
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
	#backupRestoreTask .editButton i{
		font-size:14px !important;
	}
	#backupRestoreTask .editButton span{
		font-size:12px;
	}
	#backupRestoreTask .inputCommon .el-input__inner { 
		border-radius: 4px; 
	}
	#backupRestoreTask .validateItem .el-input-group__append { 
		border:none; 
		background:none; 
		padding: 0px 10px; 
	}
	#backupRestoreTask .validateItem .el-input__inner { 
		width:186px; 
	}
	#backupRestoreTask .is-error .el-input-group__append{ 
		color:#FA5555; 
	}
	#backupRestoreTask .validateItem .el-input-group__append { 
		border:none; background:none; 
	}
	#backupRestoreTask .validateItem .el-form-item__error { 
		display:none; 
	}
	#backupRestoreTask .el-form-item__error {
		margin-left: 0;
	}
	/* Time 字段上下结构样式 */
	#backupRestoreTask .timeFieldTop .el-form-item__label {
		width: auto !important;
		text-align: left !important;
		margin-bottom: 5px;
		display: block;
	}
	#backupRestoreTask .timeFieldTop .el-form-item__content {
		margin-left: 0 !important;
	}
	.selectDateItem {
		font-size: 12px;
		font-weight: normal;
		transition: all 0.3s ease;
		text-align: center;
		cursor: pointer;
		border-radius: 50%;
	}
	#backupRestoreTask .commonWidthItem .el-input {
		width: 300px;
	}
	/* FTP Server 样式 */
	#backupRestoreTask .executeModeWarp {
		padding-bottom: 50px;
	}
	#backupRestoreTask .executeModeWarp .executeModeInfor {
		padding: 30px 50px 0;
	}
	#backupRestoreTask .ftpServerContainer {
		padding: 20px 0 0 76px;
		display: flex;
		flex-wrap: wrap;
		gap: 0 180px;
		max-width: 800px;
	}
	#backupRestoreTask .ftpServerContainer .el-form-item {
		flex: 0 0 300px;
	}
	#backupRestoreTask .ftpServerContainer .el-input,
	#backupRestoreTask .ftpServerContainer .el-password {
		width: 300px;
	}
	#backupRestoreTask .ftpServerContainer .el-input[style*="display: none"] {
		display: none !important;
	}
	
	/* 执行模式信息标题样式 */
	#backupRestoreTask .executeModeInfor.group-title {
		padding: 30px 50px 0;
	}
	/* 定时执行单选按钮样式 */
	#backupRestoreTask .executeModeBox .el-radio[label="timing"] {
		margin-bottom: 0px;
		margin-left: 80px;
	}
	/* Execute 标签必填星号样式 */
	#backupRestoreTask .execute-label-required {
		color: #FF4614;
	}
	
	/* Week 选择器 Popover 内容 */
	#backupRestoreTask .week-popover-content {
		padding: 10px 20px;
	}
	#backupRestoreTask .week-days-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 8px;
	}
	/* Month 选择器 Popover 内容 */
	#backupRestoreTask .month-popover-content {
		padding: 15px;
		background: #F9FAFC;
		border-radius: 8px;
	}
	#backupRestoreTask .month-popover-title {
		margin-bottom: 10px;
		font-size: 13px;
		color: #606266;
		font-weight: bold;
	}
	#backupRestoreTask .month-days-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 10px 20px;
	}
	/* 选择器输入框指针样式 */
	#backupRestoreTask .selector-input-readonly {
		cursor: pointer;
	}
</style>

<!--备份与恢复：新建普通备份，恢复任务、 新建或修改周期备份任务-->
<div id="backupRestoreTask">
	<div class="newBackupTaskWarp">
		<el-form :model='ruleForm' :rules="rules" ref="ruleForm">
			<!-- 基本信息：普通备份任务或恢复任务 -->
			<div v-if="addTaskType == 'backup' || addTaskType == 'restore'" class="basicInfoWarp" >
				<div class="group-title not-extend basicInfo">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
				</div>
				<el-form-item class="newTaskName" label='<%=rb.getString("RenWuMingCheng")%>' prop='taskName'>
					<el-input maxlength="100" v-model="ruleForm.taskName"></el-input>
				</el-form-item>
				<el-form-item class="newTaskName" label='<%=rb.getString("HuiFuLeiXing")%>' prop='restoreType' v-if="addTaskType == 'restore'">
					<div class="boxBorderCon">
						<el-radio-group v-model='restoreType'>
							<el-radio label="restore"><%=rb.getString("HuiFuDaoZuiXinGengXinDeWenJianPeiZhi")%></el-radio>
							<el-radio label="reset" style="margin-left:40px;"><%=rb.getString("HuiFuChuChangPeiZhi")%></el-radio>
						</el-radio-group>
					</div>
				</el-form-item>
			</div>
			<!--周期备份任务开关-->
			<div v-else>
				<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='enable' label-width='40px' class="executeModeBox" style="font-weight: bold;margin-bottom: 0; padding: 20px 0 0 54px;">
					<el-switch v-model="ruleForm.enable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
			</div>
			<!--设备选择  -->
			<div class="selectDeviceWarp">
				<div class="group-title not-extend selectDeviceInfo">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
				</div>
				<el-form-item class="newTaskName" label='<%=rb.getString("SheBeiZhiDing")%>'>
                    <el-radio-group v-model="deviceType">
                        <el-radio label="1" border><%=rb.getString("QuanBu")%></el-radio>
                        <el-radio label="2" border><%=rb.getString("ZhiDingZhiXing")%></el-radio>
                    </el-radio-group>
                </el-form-item>
				<div class='selectDeviceBox'>
					<div style='width:100%;'>			
						<div style='display:flex;height:300px;width:100%; '>
							<div style='flex:1'>
								<el-pairgrid v-show="deviceTypeSpecified"
									:id="'selectDeviceList'" 
									:rownumber="true" 
									ref="backupRestoreTaskTable"  
									@selection-change='selectChange' 
									:right-url="rightUrl"
									:left-url="leftUrl" :height="height" 
									row-key="small_cell_code" 
									:query-params="deviceSelectParams"										
									:title="deviceTitle" 
									:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
									<template slot="left">
										<el-table-column type='selection' width="45"></el-table-column>
										<el-table-column prop="connection_status" width="50">
											<template slot-scope="scope">
												<div :class="{
													'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
													'':scope.row.have_connected==2,
													'conn_exc':scope.row.connection_status=='Exception',
													'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
											</template>
										</el-table-column>
										<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" sortable></el-table-column>
										<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" sortable></el-table-column>
										<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' prop="product_type" sortable></el-table-column>

										<el-table-column label='<%=rb.getString("ZuiXinGengXinWenJian")%>' prop="latest_update_file"></el-table-column>
										<el-table-column label='<%=rb.getString("ZuiXianGengXinShiJian")%>' prop="latest_update_time" sortable></el-table-column>
									</template>
									<!-- 模糊查询 -->
									<template slot="toolbar">
					               		<div class="searchCon commonFlex" style="position: relative;">
					               		    <span class="commonText14" style='line-height: 28px; margin-right: 20px;'><%=rb.getString("ENBSheBei")%></span>
											<el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="specifiedDevicesSearch">
												<i slot="suffix" class="el-icon el-icon-common-search"  @click="deviceSelectQuery"></i>
											</el-input>
											<div class="tableHeadQueryBoxCls" style='padding: 0 10px;'>
                                                <div v-for="(item,index) in enbAdvancedQueryItemList" style='margin: -5px 0 0;'>
                                                    <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															:label='item.label'
															v-model="item.checkedItemList"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
														</el-popfilter>
													</div>
													<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															type="single"
															:label='item.label'
															v-model="item.selectVal"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.selectVal)">
														</el-popfilter>
													</div>
                                                </div>
                                                <div class="advancedQueryItemBox"  style="background: #FFF;" @click="enbClearFilterClick">
                                                    <%=rb.getString("QingKongShaiXuan")%>
                                                </div>
												<div class="editButton" @click="batchBtnClick" size="mini">
													<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
													<span><%=rb.getString("PiLiangShuRu")%></span>
												</div>
                                            </div>
										</div>
					                </template>	
									<!--右侧的下拉表格 -->
									<template slot='right'>
										<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
										<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
									</template>
								</el-pairgrid>
									
								<el-ctable v-if="deviceTypeAll" 
									ref="backupRestoreTaskTableAll" 
									id="selectDeviceListAll" 
									row-key="small_cell_code" 
									:url="deviceUrl" 
									:height="height" 
									:query-params="paramsAll" 
									pagination="true"
									style="border:1px solid #E9E9E9;height:300px;">
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" sortable></el-table-column>
									<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" sortable></el-table-column>
									<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' prop="product_type" sortable></el-table-column>
									<el-table-column label='<%=rb.getString("ZuiXinGengXinWenJian")%>' prop="latest_update_file"></el-table-column>
									<el-table-column label='<%=rb.getString("ZuiXianGengXinShiJian")%>' prop="latest_update_time" sortable></el-table-column>
									<!-- 模糊查询 -->
									<template slot="toolbar">
					               		<div class="searchCon commonFlex">
											<el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="allDevicesSearch">
												<i slot="suffix" class="el-icon el-icon-common-search"  @click="deviceQueryAll"></i>
											</el-input>
											<div class="tableHeadQueryBoxCls" style='padding: 0 20px;'>
                                                <div v-for="(item,index) in enbAdvancedQueryItemList" style='margin: 0;'>
                                                    <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															:label='item.label'
															v-model="item.checkedItemList"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
														</el-popfilter>
													</div>
													<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															type="single"
															:label='item.label'
															v-model="item.selectVal"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.selectVal)">
														</el-popfilter>
													</div>
                                                </div>
                                                <div class="advancedQueryItemBox"  style="background: #FFF;" @click="enbClearFilterClick">
                                                    <%=rb.getString("QingKongShaiXuan")%>
                                                </div>
                                            </div>
										</div>
					                </template>	
								</el-ctable>
							</div>
						</div>											
					</div>					
				</div>
				<el-form-item prop='cellCodes' class="noSelectDevice">
					<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
				</el-form-item>
			</div>
			
			<!-- 执行模式： 普通备份任务或恢复任务 -->
			<div class="selectDeviceWarp">
				<div class="group-title not-extend executeModeInfor">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
				</div>
				<div v-if="addTaskType == 'backup' || addTaskType == 'restore'" class="executeModeBox" >
					<el-form-item prop='executeMode'>
						<el-radio-group v-model="ruleForm.executeMode">
							<el-radio label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
							<el-radio label="timing" class="timing-radio"><%=rb.getString("DingShiZhiXing")%></el-radio>
						</el-radio-group>
					</el-form-item>
					<el-form-item prop='startTime' class='timeSelect'>
						<el-date-picker 
							value-format="yyyy-MM-dd HH:mm:ss" 
							v-model='ruleForm.startTime' 
							:disabled="setTimeEnable" 
							type="datetime" 
							@focus='setTime' 
							:picker-options="pickerOptions">
						</el-date-picker>	
					</el-form-item>
				</div>

				<!--周期备份任务： 根据执行默认校验字段必填-->
				<div v-else class="ftpServerContainer" style="gap: 0 180px;">
					<!-- Execute 下拉框 -->
					<el-form-item label='<%=rb.getString("ZhouQiZhiXing")%>' prop='executeMode' label-position="top" class="commonWidthItem">
						<el-select v-model='ruleForm.executeMode'>
							<el-option label='<%=rb.getString("Tian")%>' value="every day"></el-option>
							<el-option label='<%=rb.getString("Zhou")%>' value="every week"></el-option>
							<el-option label='<%=rb.getString("Yue")%>' value="every month"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item v-if="ruleForm.executeMode == 'every week'" label='<%=rb.getString("Zhou")%>' prop='periodTimeWeek' class="commonWidthItem" label-position="top">
						<el-popover ref="weekPopover"
							placement="bottom-start"
							width="380"
							trigger="click"
							popper-class="week-day-popover">
							<div style="padding: 16px 20px;">
								<div style="display:grid;grid-template-columns:repeat(7,1fr);gap:10px 15px;">
									<div v-for="week in weekDays" :key="week.value" class="selectDateItem" @click="toggleWeekDay(week.value)"
										:style="{
											width: '32px',
											height: '32px',
											lineHeight: '32px',	
											backgroundColor: ruleForm.weekDay.includes(week.value) ? '#4D84FF' : '#FFFFFF',
											color: ruleForm.weekDay.includes(week.value) ? '#FFF' : '#606266',
										}"
										@mouseover="$event.currentTarget.style.transform='translateY(-2px)'"
										@mouseout="$event.currentTarget.style.transform='translateY(0)'">
										{{week.label}}
									</div>
								</div>
							</div>
							<el-input slot="reference" 
								:value="getWeekDayLabels()"
								placeholder=""
								prefix-icon="el-icon-date"
								readonly
								style="cursor:pointer;">
							</el-input>
						</el-popover>
					</el-form-item>

					<!-- Month 月份选择器 -->
					<el-form-item v-if="ruleForm.executeMode == 'every month'" label='<%=rb.getString("Yue")%>' prop='periodTimeMonth' class="commonWidthItem" label-position="top">
						<el-popover ref="monthSelectPopover"
							placement="bottom-start"
							width="300"
							trigger="click"
							popper-class="month-select-popover">
							<div style="padding: 20px;">
								<div style="display:grid;grid-template-columns:repeat(6,1fr);gap:10px 15px;">
									<div v-for="month in 12" :key="month" class="selectDateItem"
										@click="toggleMonth(month)"
										:style="{
											width: '30px',
											height: '30px',
											lineHeight: '30px',
											backgroundColor: ruleForm.months.includes(month + '') ? '#4D84FF' : '#FFFFFF',
											color: ruleForm.months.includes(month + '') ? '#FFFFFF' : '#606266',
										}"
										@mouseover="$event.currentTarget.style.transform='scale(1.05)'"
										@mouseout="$event.currentTarget.style.transform='scale(1)'">
										{{month}}
									</div>
								</div>
							</div>
							<el-input slot="reference" 
								:value="ruleForm.months.length > 0 ? ruleForm.months.map(m => m).join(', ') : ''"
								placeholder=""
								prefix-icon="el-icon-date"
								readonly
								style="cursor:pointer;">
							</el-input>
						</el-popover>
					</el-form-item>

					<!-- Month 选择器 -->
					<el-form-item v-if="ruleForm.executeMode == 'every month'" label='<%=rb.getString("Tian")%>' prop='periodTimeMonthDay' class="commonWidthItem" label-position="top">
						<el-popover ref="monthDayPopover"
							placement="bottom-start"
							width="398"
							trigger="click"
							popper-class="month-day-popover">
							<div style="padding: 20px;">
								<div style="display:grid;grid-template-columns:repeat(8,1fr);gap:10px 15px;">
									<div v-for="day in 31" :key="day" class="selectDateItem"
										@click="toggleMonthDay(day)"
										:style="{
											width: '30px',
											height: '30px',
											lineHeight: '30px',
											backgroundColor: ruleForm.monthDay.includes(day + '') ? '#4D84FF' : '#FFFFFF',
											color: ruleForm.monthDay.includes(day + '') ? '#FFFFFF' : '#606266',
										}"
										@mouseover="$event.currentTarget.style.transform='scale(1.1)'"
										@mouseout="$event.currentTarget.style.transform='scale(1)'">
										{{day}}
									</div>
								</div>
							</div>
							<el-input slot="reference" 
								:value="ruleForm.monthDay.length > 0 ? ruleForm.monthDay.join(', ') : ''"
								placeholder=""
								prefix-icon="el-icon-date"
								readonly
								style="cursor:pointer;">
							</el-input>
						</el-popover>
					</el-form-item>

					<!-- Time 时间选择器 -->
					<el-form-item label='<%=rb.getString("ShiJian")%>' prop='periodTime' class="commonWidthItem" label-position="top">
						<el-time-picker 
							ref="periodTimePicker"
							value-format="HH:mm:ss" 
							v-model='ruleForm.periodTime' 
							@focus="defaultStartTimes"
							:picker-options="pickerOptions">
						</el-time-picker>
					</el-form-item>
				</div>
			</div>

			<!-- ftp server -->
			<div v-if="addTaskType == 'periodBackup' || addTaskType == 'updatePeriodBackup'" class="executeModeWarp" >
				<div class="group-title not-extend executeModeInfor">
					<span class="title-icon"></span>
					<span class="title-text">FTP Server</span>
				</div>
				<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='ftpSwitch' label-width='40px' class="executeModeBox">
					<el-switch v-model="ruleForm.ftpSwitch" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<div class="ftpServerContainer" style="padding: 0 0 0 76px;">
					<el-form-item label='<%=rb.getString("FTPXieYi")%>' prop="ftpProtocol" label-position="top" class="commonWidthItem">
						<el-select v-model="ruleForm.ftpProtocol" class="language">
							<el-option label='SFTP' value="sftp"></el-option>
							<el-option label='FTP' value="ftp"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item label='<%=rb.getString("ShangChuanLuJing")%>' prop="ftpPath" label-position="top">
						<el-input v-model="ruleForm.ftpPath"></el-input>
					</el-form-item>
					
					<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop="ftpIp" label-position="top">
						<el-input v-model="ruleForm.ftpIp"></el-input>
					</el-form-item>
					<el-form-item label='<%=rb.getString("DuanKou")%>' prop="ftpPort" label-position="top">
						<el-input v-model="ruleForm.ftpPort"></el-input>
					</el-form-item>
					
					<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="ftpUser"  label-position="top">
						<el-input v-model="ruleForm.ftpUser" maxlength="60"></el-input>
					</el-form-item>
					<el-form-item label='<%=rb.getString("MiMa")%>' prop="ftpPassword"  label-position="top">
						<el-password v-model="ruleForm.ftpPassword" size="mini" show-password placeholder=""></el-password>
						<el-input v-model="ruleForm.ftpPassword" style="display: none;"></el-input>
					</el-form-item>
				</div>
			</div>	
		</el-form>
	</div>
	<!--batch input-->
	<el-dialog class='dialogStyle' title='<%=rb.getString("TianJia")%>' width='630px' :visible.sync='batchSnDialog' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='batchSnForm' :rules='batchSnRules' :model='batchSnForm' label-position="top">
			<div>
				<label><%=rb.getString("Title_SheBeiBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='batchSnForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<span class="commonNotes12"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
/*
周期备份任务-2025-12-03：
1、新增、修改周期备份任务：选择设备、执行模式等模块中的必填项需根据周期备份开关进行校验：
	开关-打开： 选择设备、执行模式校验必填；
	开关-关闭： 不校验必填；
2、如含有周期备份任务：
	可进行修改任务；实际无查看周期备份任务功能；修改和查看属于同一模式；
3、FTP Server 模块必填校验：
	周期备份开关控制 && FTP Server 开关 双开关打开，校验必填；
	反之，只会校验字段是否合法；
	
4、addTaskType 变量说明：
	backup：新建备份任务
	restore：新建恢复任务
	periodBackup: 新建周期备份任务
	updatePeriodBackup: 修改周期备份任务

*/
var addModifyBackupRestore = new Vue({
	el:'#backupRestoreTask',
	data(){
		var vm = this,
			//只有back up和restore任务类型时，任务名称才必填
			validateName = (rule,value,callback) => {
				if(vm.addTaskType == 'backup' || vm.addTaskType == 'restore'){
					if(value === ''){
						callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			//只有back up和restore任务类型时,必填
			validateStartTime = (rule,value,callback) => {
				if(vm.addTaskType == 'backup' || vm.addTaskType == 'restore'){
					if(vm.ruleForm.executeMode !== 'timing'){
						callback()
					}else{
						if(value === '' || value === null){
							callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
						}else{
							callback();
						}
					}
				}else{
					callback();
				}
			},

			validateCodes = (rule,value,callback) => {
				if(vm.deviceType == '1'){
					callback()
				}else{
					if(vm.addTaskType == 'backup' || vm.addTaskType == 'restore'){
						if(value == '' || value == null){
							callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
						}else{
							callback();
						}
					}else{
						if(vm.ruleForm.enable == '1'){
							if(value == '' || value == null){
								callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
							}else{
								callback();
							}
						}else{
							callback();
						}
					}
				}
			},
			validatePeriodTime = (rule,value,callback) => {		
				if(vm.addTaskType == 'backup' || vm.addTaskType == 'restore'){
					callback();
				}else{
					if(vm.ruleForm.enable == '1'){
						if(value === '' || value === null){
							callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
						}else{
							callback();
						}
					}else{
						callback();
					}
				}	
			},
			// FTP字段条件校验：仅当ftpSwitch 和 periodSwitch均打开时，校验必填；否则只校验格式
			validateFtpPath = (rule,value,callback) => {
				if(vm.ruleForm.enable == '1' && vm.ruleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuShangChuanLuJing")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateFtpIp = (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				// 开关打开时，校验必填
				if(vm.ruleForm.enable == '1' && vm.ruleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("XinIPShuRuTiShi")%>'))
						return;
					}
				}
				
				// 无论开关状态，如果有值则校验IP格式
				if(value && value != ''){
					if(!reg.test(value)){
						callback(new Error('<%=rb.getString("XinIPGeShiCuoWu")%>'))
						return;
					}
				}
				
				callback();
			},
			validateFtpPort = (rule,value,callback) => {
				var regNum = /^\d+$/;
				
				// 开关打开时，校验必填
				if(vm.ruleForm.enable == '1' && vm.ruleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuDuanKou")%>'))
						return;
					}
				}
				
				// 无论开关状态，如果有值则校验端口格式和范围（0-65535）
				if(value && value != ''){
					var portNum = parseInt(value);
					if(!regNum.test(value) || (value < 0 || value > 65535)){ 
						callback(new Error('<%=rb.getString("ChangDuChaoChuFanWei")%>'))
						return;
					}
				}
				
				callback();
			},
			validateFtpUser = (rule,value,callback) => {
				if(vm.ruleForm.enable == '1' && vm.ruleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuYongHuMing")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateFtpPassword = (rule,value,callback) => {
				if(vm.ruleForm.enable == '1' && vm.ruleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuMiMa")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			// 周选择器校验：executeMode为week时必选
			validateWeekDay = (rule,value,callback) => {
				if(vm.ruleForm.enable == '1' && vm.ruleForm.executeMode == 'every week'){
					if(!vm.ruleForm.weekDay || vm.ruleForm.weekDay.length == 0){
						callback(new Error('<%=rb.getString("QingZhiShaoXuanZeYiTian")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			// 月选择器校验：executeMode为month时必选
			validateMonthDay = (rule,value,callback) => {
				if(vm.ruleForm.enable == '1' && vm.ruleForm.executeMode == 'every month'){
					if(!vm.ruleForm.monthDay || vm.ruleForm.monthDay.length == 0){
						callback(new Error('<%=rb.getString("QingZhiShaoXuanZeYiTian")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			// 月份选择器校验：executeMode为month时必选
			validateMonth = (rule,value,callback) => {
				if(vm.ruleForm.enable == '1' && vm.ruleForm.executeMode == 'every month'){
					if(!vm.ruleForm.months || vm.ruleForm.months.length == 0){
						callback(new Error('<%=rb.getString("QingZhiShaoXuanZeYiYue")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validatorSn = (rule,value,callback) => {
				var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
					list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
						return item.length > 0;
					});
		
				if (serialNumber == null || serialNumber.length == 0) {
					callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
				} else {
					var nameFlag = list.every(function(item,index){
						return temp.test(item)
					})
					if(nameFlag){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}
			};
		return {
			leftUrl:'',
			rightUrl:'',
			deviceTitle:['<%=rb.getString("BackupRestoreSheBeiLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			height:'300px',
			setTimeEnable:true,
			restoreType:'restore',
			ruleForm:{
				taskName:'${addTaskName}',
				taskType:"",
				executeMode:'active',
				startTime:'',
				timeZone:timeZone,
				cellCodes:'',

				enable: '0',
				periodTime:'',
				periodTimeWeek: '',
				weekDay: [],
				periodTimeMonth: '',
				periodTimeMonthDay: '',
				monthDay: [],
				months: [],

				ftpSwitch: '0',
				ftpUser: '',
				ftpIp: '',
				ftpPort: '',
				ftpPath: '',
				ftpPassword: '',
				ftpProtocol: 'sftp',
				cronPeriod: '' //传 cron表达式
			},
			paramsAll:{
				timeZone: timeZone,
				product_type:'',
				searchText:'',
				likeFields: 'serial_number,host_name'
			},
			deviceSelectParams:{
				timeZone: timeZone,
				product_type:'',
				searchText:'',
				likeFields: 'serial_number,host_name'
			},
			filterParams: {
				likeFields: 'serial_number,host_name'
			},
			product_type:'',
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			deviceUrl:'',	
			deviceType:'2',
			addTaskType:'',
			weekDays: [
				{label: 'Sun', value: '1'},
				{label: 'Mon', value: '2'},
				{label: 'Tue', value: '3'},
				{label: 'Wed', value: '4'},
				{label: 'Thu', value: '5'},
				{label: 'Fri', value: '6'},
				{label: 'Sat', value: '7'}
			],
			selection:[],
			updateTaskId:'',
			rules:{
				//back up, restore 类型任务：taskName、cellCodes、executeMode、startTime 等字段校验必填；
				taskName: [
					{ validator: validateName, trigger:'blur' }
				],
				//back up, restore 类型的任务才会配置该字段
				startTime: [
					{ type:'date', validator: validateStartTime, trigger: 'change' }
				],
				// cellCodes、executeMode、periodTime、periodTimeWeek、periodTimeMonth、periodTimeMonthDay 等字段需根据 周期备份开关状态决定是否校验必填；
				cellCodes:[
					{ validator: validateCodes, trigger: 'change' }
				],				
				executeMode: [
					{ required: true, message: '<%=rb.getString("QingXuanZeZhiXingFangShi")%>', trigger: 'change' }
				],
				//只有周期备份任务才会配置以下字段块；
				periodTime: [
					{required: true, type: 'date', validator: validatePeriodTime, trigger: 'change' }
				],
				periodTimeWeek: [
					{ required: true, validator: validateWeekDay, trigger: 'change' }
				],
				periodTimeMonth: [
					{ required: true, validator: validateMonth, trigger: 'change' }
				],
				periodTimeMonthDay: [
					{ required: true, validator: validateMonthDay, trigger: 'change' }
				],
				//只有周期备份任务才会配置该模块； 受周期备份开关和 FTP Server 开关控制的必填校验；
				ftpProtocol: [
					{ required: true, message: '<%=rb.getString("QingXuanZeFTPXieYi")%>', trigger: 'change' }
				],
				ftpPath: [
					{ required: true, validator: validateFtpPath }
				],
				ftpIp: [
					{ required: true, validator: validateFtpIp }
				],
				ftpPort: [
					{ required: true, validator: validateFtpPort }
				],
				ftpUser: [
					{ required: true, validator: validateFtpUser }
				],
				ftpPassword: [
					{ required: true, validator: validateFtpPassword }
				],
			},
			allDevicesSearch:'',
			specifiedDevicesSearch:'',
			deviceTypeSpecified:true,
			deviceTypeAll:false,
			enbAdvancedQueryItemList:[
                {
                    type:'checkbox',
                    isShow:true,
                    isIndeterminate:false,
                    checkAll:false,
                    popoverShow:false,
                    checkedItemList:[],
                    oldCheckedItemList:[],
                    label:'<%=rb.getString("ChanPinLeiXing") %>',
                    options:[],
                    value:'product_type',
                },
            ],
			//batch add
			batchSnDialog: false,
			batchSnForm:{
				serialNumber: '',
				type: 'input'
			},
			batchSnRules:{
				serialNumber:[
					{validator: validatorSn, trigger:'change'}
				]
			},
		}
	},
	watch:{
		restoreType:function(val){
			var vm = this;

			if(vm.addTaskType == 'restore'){
				vm.deviceType = '2';

				if(val == 'reset'){	
					vm.specifiedDevicesSearch = '';
					vm.deviceSelectParams.searchText = '';				
					vm.deviceSelectParams.isFilterAuxiliary = '1';
					//选中all devices perform 时
					vm.allDevicesSearch = '';
					vm.paramsAll.searchText = '';
					vm.paramsAll.isFilterAuxiliary = '1';
				}else{
					//全部设备
					vm.allDevicesSearch = '';
					vm.paramsAll.searchText = '';
					delete vm.paramsAll.isFilterAuxiliary;
					//已选设备
					vm.specifiedDevicesSearch = '';
					vm.deviceSelectParams.searchText = '';
					delete vm.deviceSelectParams.isFilterAuxiliary;
				}
				
				vm.$nextTick(function(){
					
					if(vm.$refs.backupRestoreTaskTable){
						vm.$refs.backupRestoreTaskTable.reload();
					}
				});
			}						
		},
		deviceType:function(val){
			var vm = this;

			if(val == '1'){
				vm.deviceTypeAll = true;
				vm.deviceTypeSpecified = false;
				vm.specifiedDevicesSearch = '';
				vm.deviceSelectParams.searchText = '';
				vm.deviceSelectParams.product_type = '';
				if(vm.$refs.ruleForm){
					vm.$refs.ruleForm.clearValidate('cellCodes');
				}
			}else{
				vm.deviceTypeSpecified = true;
				vm.deviceTypeAll = false;
				vm.allDevicesSearch = '';
				vm.paramsAll.searchText = '';
				vm.paramsAll.product_type = '';
			}

			vm.enbClearFilterClick();
		},
		selection(){
			var vm = this,
				data = vm.$refs.backupRestoreTaskTable.getData();
				
			if(data.length != 0){
				vm.ruleForm.cellCodes = data.map(function(item){
					return item.small_cell_code;
				}).join(',');
			}else{
				vm.ruleForm.cellCodes = '';
			}
		},
	
		"ruleForm.executeMode":function(newVal){
			var vm = this;			
			if(vm.addTaskType == 'backup' || vm.addTaskType == 'restore'){
				// 根据 executeMode 的值设置时间选择器的启用/禁用状态
				if(newVal == 'timing'){
					vm.setTimeEnable = false;
				}else{
					vm.setTimeEnable = true;
				}
				vm.$refs.ruleForm.validateField('startTime')
			}else{
				vm.$refs.weekPopover && vm.$refs.weekPopover.doClose();
				vm.$refs.monthSelectPopover && vm.$refs.monthSelectPopover.doClose();
				vm.$refs.monthDayPopover && vm.$refs.monthDayPopover.doClose();
				vm.$refs.periodTimePicker && vm.$refs.periodTimePicker.hidePicker();

				vm.ruleForm.weekDay = [];
				vm.ruleForm.monthDay = [];
				vm.ruleForm.months = [];
				vm.ruleForm.periodTime = '';
				
				// 清除验证状态
				if(vm.$refs.ruleForm){
					vm.$nextTick(function(){
						vm.$refs.ruleForm.clearValidate(['periodTime', 'periodTimeWeek', 'periodTimeMonth', 'periodTimeMonthDay']);
					});
				}
			}	
		},
		"ruleForm.ftpSwitch":function(newVal){
			// 当FTP开关状态改变时，清除相关字段的验证状态
			var vm = this; 
			if(vm.$refs.ruleForm){
				vm.$nextTick(function(){
					// 清除FTP相关字段的验证提示
					vm.$refs.ruleForm.clearValidate(['ftpProtocol','ftpPath','ftpIp','ftpPort','ftpUser','ftpPassword']);
				});
			}
		},
	},
	methods:{ 
		init(){
			var vm = this;

			vm.$nextTick(function(){
				vm.leftUrl = '${ctx}/task/enb/config/backupRestore/queryCellInfos.action';	
				vm.deviceUrl = '${ctx}/task/enb/config/backupRestore/queryCellInfos.action';
				
				// 确保在 nextTick 中访问 ref，避免 ref 未初始化
				if(vm.$refs.backupRestoreTaskTable && eNBBackupRestoreTasks.deviceSnSelection){
					vm.$refs.backupRestoreTaskTable.appendCheckedRows(eNBBackupRestoreTasks.deviceSnSelection);
				}
			});

			var params={
                likeFields: 'serial_number,host_name'
            };
            axios.post( '${ctx}/task/enb/config/backupRestore/getProductType.action', stringify(params)).then(function(response){
                var data = response.data;
                
                var arr = [];
                (data.product_type || []).map(function(item){
                    if (item){
                        arr.push({label:item.text,value:item.value})
                    }
                })
                vm.enbAdvancedQueryItemList.map((items)=>{
                    if('product_type' == items.value){
                        items.options = arr;
                    }
                })
                 
            });
		},
		initAddTask(type, data){
			var vm = this;
			
			vm.addTaskType = type;	
			vm.updateTaskId = data.task_id;
			
			if(type == 'updatePeriodBackup'){		
				if(data.selected_type == 'all'){
					vm.deviceType = '1';
				}else{
					//指定设备执行
					vm.deviceType = '2';				
					vm.rightUrl = '${ctx}/task/enb/config/backupRestore/periodTask/deviceList.action?taskId=' + vm.updateTaskId;		
				}
				// 周期备份开关
				vm.ruleForm.enable = data.is_enable == 1 ? '1' : '0'; 
				// 回显 FTP Server 数据
				vm.ruleForm.ftpSwitch = data.ftpSwitch || '0';
				vm.ruleForm.ftpUser = data.ftpUser || '';
				vm.ruleForm.ftpIp = data.ftpIp || '';
				vm.ruleForm.ftpPort = data.ftpPort || '';
				vm.ruleForm.ftpPath = data.ftpPath || '';
				vm.ruleForm.ftpPassword = data.ftpPassword || '';
				vm.ruleForm.ftpProtocol = data.ftpProtocol || 'sftp';
				
				// 回显执行模式相关数据
				vm.ruleForm.executeMode = data.executeMode || 'every day';
			}								
			vm.$nextTick(function(){		
				// 根据任务类型设置 executeMode 默认值（在 nextTick 中设置，确保表单已初始化）
				if(type == 'periodBackup'){
					// 新建周期备份任务，默认为 天
					vm.ruleForm.executeMode = 'every day';
				}else if(type == 'updatePeriodBackup'){
					// 新建周期备份任务，默认为 天
					vm.ruleForm.executeMode = data.executeMode || 'every day';

					// periodTime 始终从 cronPeriod 中解析（在 nextTick 中执行，确保 DOM 已更新）
					if(data.cronPeriod){
						vm.parseCronExpression(data.cronPeriod, data.executeMode);
					}
				}else{
					// 普通备份/恢复任务，默认为 active
					vm.ruleForm.executeMode = 'active';
				}
				
				initForm(vm.$refs.ruleForm);
	    	});
		},	
		//天，周，月： 秒 分 时 日 月 周
		getCronExpression(){
			var vm = this,
				time = vm.ruleForm.periodTime || '00:00:00',
				timeParts = time.split(':'),
				second = timeParts[2] || '0',
				minute = timeParts[1] || '0',
				hour = timeParts[0] || '0',
				cronPeriod = '';
			
			switch(vm.ruleForm.executeMode){
				case 'every day':
					// day: 秒 分 时 日 月 周 
					cronPeriod = second + ' ' + minute + ' ' + hour + ' * * ?';
					break;
					
				case 'every week':
					// week: 秒 分 时 日 月 周
					// 例如: 10 10 10 ? * 1,3,5 (每周一、三、五10:10:10执行)
					var weekDays = vm.ruleForm.weekDay.length > 0 ? vm.ruleForm.weekDay.join(',') : '*';
					cronPeriod = second + ' ' + minute + ' ' + hour + ' ? * ' + weekDays;
					break;  
					
				case 'every month':
					// month: 秒 分 时 日 月 周 
					// 月份在第5位，日期在第4位
					var monthDays = vm.ruleForm.monthDay.length > 0 ? vm.ruleForm.monthDay.join(',') : '*';
					var months = vm.ruleForm.months.length > 0 ? vm.ruleForm.months.join(',') : '*';
					cronPeriod = second + ' ' + minute + ' ' + hour + ' ' + monthDays + ' ' + months + ' ?';
					break;
					
				default:
					cronPeriod = second + ' ' + minute + ' ' + hour + ' * * ?';
			}
			
			return cronPeriod;
		},
		// 解析 cron 表达式并回显到页面
		parseCronExpression(cronPeriod, executeMode){
			var vm = this;
			
			// 如果没有 cronPeriod，直接返回
			if(!cronPeriod){
				return;
			}
			
			// 解析 cron 表达式: 秒 分 时 日 月 周 年
			var cronParts = cronPeriod.trim().split(/\s+/);
			
			if(cronParts.length < 6){
				return;
			}
			
			var second = cronParts[0] || '0',
				minute = cronParts[1] || '0',
				hour = cronParts[2] || '0',
				dayOfMonth = cronParts[3] || '*',
				month = cronParts[4] || '*',
				dayOfWeek = cronParts[5] || '?';
			
			// 格式化时间为两位数（例如：09:05:03）
			var formatTime = function(val){
				return String(val).length === 1 ? '0' + val : String(val);
			};
			
			// 回显时间（时:分:秒）- 始终从 cron 表达式中解析
			var timeStr = formatTime(hour) + ':' + formatTime(minute) + ':' + formatTime(second);
						
			// 使用 Vue.set 确保响应式更新
			vm.$set(vm.ruleForm, 'periodTime', timeStr);
			

			// 根据 executeMode 回显对应的日期选择
			if(executeMode === 'every week'){
				// 解析周几（dayOfWeek）
				if(dayOfWeek && dayOfWeek !== '?' && dayOfWeek !== '*'){
					vm.ruleForm.weekDay = dayOfWeek.split(',');
				}
			}else if(executeMode === 'every month'){
				// 解析每月的日期（dayOfMonth）和月份（month）
				if(dayOfMonth && dayOfMonth !== '?' && dayOfMonth !== '*'){
					vm.ruleForm.monthDay = dayOfMonth.split(',');
				}
				if(month && month !== '?' && month !== '*'){
					vm.ruleForm.months = month.split(',');
				}
			}
		},
		toggleWeekDay(dayValue){
			var vm = this,
				index = vm.ruleForm.weekDay.indexOf(dayValue);

			if(index > -1){
				vm.ruleForm.weekDay.splice(index, 1);
			}else{
				vm.ruleForm.weekDay.push(dayValue);
			}
			
			// 触发验证
			vm.$nextTick(function(){
				if(vm.$refs.ruleForm){
					vm.$refs.ruleForm.validateField('periodTimeWeek');
				}
			});
		},
		getWeekDayLabels(){
			var vm = this;

			if(vm.ruleForm.weekDay.length === 0) return '';

			var labels = vm.ruleForm.weekDay.map(function(val){
				var day = vm.weekDays.find(function(d){ return d.value === val; });
				return day ? day.label : '';
			});
			return labels.join(', ');
		},
		toggleMonth(month){
			var vm = this;
				monthStr = month + '',
				index = vm.ruleForm.months.indexOf(monthStr);

			if(index > -1){
				vm.ruleForm.months.splice(index, 1);
			}else{
				vm.ruleForm.months.push(monthStr);
			}
			
			// 触发验证
			vm.$nextTick(function(){
				if(vm.$refs.ruleForm){
					vm.$refs.ruleForm.validateField('periodTimeMonth');
				}
			});
		},
		toggleMonthDay(day){
			var vm = this;
				dayStr = day + '',
				index = vm.ruleForm.monthDay.indexOf(dayStr);

			if(index > -1){
				vm.ruleForm.monthDay.splice(index, 1);
			}else{
				vm.ruleForm.monthDay.push(dayStr);
			}
			
			// 触发验证
			vm.$nextTick(function(){
				if(vm.$refs.ruleForm){
					vm.$refs.ruleForm.validateField('periodTimeMonthDay');
				}
			});
		},
		deviceSelectQuery(val){
			var vm = this;

			vm.deviceSelectParams.searchText = vm.specifiedDevicesSearch;
		},
		deviceQueryAll(val){
			var vm = this;

			vm.paramsAll.searchText = vm.allDevicesSearch;
		},
		/**
		 * 全选操作
		 * @param selection:选择的数据
		*/
		selectChange(selection){
			var vm = this;

			vm.selection = selection;
		},
		
			
		//batch add 
		batchBtnClick(){
			var vm = this;

			vm.batchSnDialog = true;
			//清空表单
			if(vm.$refs.batchSnForm){
				vm.$refs.batchSnForm.resetFields();
			}
		},
		saveBatchSn(){
			var vm = this,  
				params = {}, 
				snStr = vm.batchSnForm.serialNumber || '',
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			params.selectedCells = list.join(",");

			vm.$refs.batchSnForm.validate((valid) => {
				if(valid){
					axios.post('${ctx}/task/enb/config/backupRestore/queryCellInfos.action', stringify(params)).then((res)=>{
						var data = res.data;
						
						if(data && data.length > 0){		
							vm.$refs.backupRestoreTaskTable.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>');
						}							
					})
				}
			})
		},
		closeBatchSn(){
			var vm = this;

			vm.batchSnDialog = false;
			vm.$refs.batchSnForm.resetFields(); 
		},

		//保存
		taskSubmit(){
	    	var vm = this, 
                message = '<%=rb.getString("ChengGong")%>';
            // 防止多次提交
            if(eNBBackupRestoreTasks.slideSubmitLoading)return

	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {},url;
	    			//新建备份或恢复任务
					if(vm.addTaskType == 'backup' || vm.addTaskType == 'restore'){
						params.taskName = vm.ruleForm.taskName;
		    			
		    			params.executeMode = vm.ruleForm.executeMode;
		    			if(vm.ruleForm.executeMode == 'timing'){
		    				params.startTime = vm.ruleForm.startTime; 
		    			}
		    			params.timeZone = timeZone;
		    			//新建恢复任务：新增 restoreType 参数
		    			if(vm.addTaskType == 'restore'){
			    			params.restoreType = vm.restoreType;
			    			//新建恢复任务类型： 
			    			if(vm.restoreType == 'restore'){
			    				params.taskType = 'restore';
			    			}else{
			    				params.taskType = 'reset';
			    			}
		    			}else{
		    				// 新建备份任务：类型
		    				params.taskType = 'backup';
		    			}
		    			
		    			if(vm.deviceType == '1'){
		    				params.selectAll = "true";
		    			}else{
		    				params.selectAll = "false";
		    				params.cellCodes = vm.ruleForm.cellCodes;
		    			}
		    			url = '${ctx}/task/enb/config/backupRestore/addBackupRestoreTask.action';	    			
						
					}else if(vm.addTaskType == 'periodBackup'){
						//新建周期备份
						params.taskName = '';
						params.periodTime = vm.ruleForm.periodTime; // 此参数无时间意义； 选择的 时间已体现在 cronPeriod 中
		    			params.timeZone = timeZone;
		    			
						params.enable = vm.ruleForm.enable;

		    			if(vm.deviceType == '1'){
		    				params.selectAll = "true";
		    			}else{
		    				params.selectAll = "false";
		    				params.cellCodes = vm.ruleForm.cellCodes;
		    			}

						params.executeMode =  vm.ruleForm.executeMode;
						params.cronPeriod = vm.getCronExpression();

						// FTP Server 和执行模式 - 解构赋值
						params.ftpSwitch = vm.ruleForm.ftpSwitch;
						params.ftpUser = vm.ruleForm.ftpUser;
						params.ftpIp = vm.ruleForm.ftpIp;
						params.ftpPort = vm.ruleForm.ftpPort;
						params.ftpPath = vm.ruleForm.ftpPath;
						params.ftpPassword = vm.ruleForm.ftpPassword;
						params.ftpProtocol = vm.ruleForm.ftpProtocol;						

						url = '${ctx}/task/enb/config/backupRestore/addOrUpdatePeriodBackupTask.action';		    			
					}else{
						//修改周期备份
						params.taskName = '';
						params.periodTime = vm.ruleForm.periodTime;
		    			params.timeZone = timeZone;
		    			params.taskId = vm.updateTaskId;

						params.enable = vm.ruleForm.enable;

		    			if(vm.deviceType == '1'){
		    				params.selectAll = "true";
		    			}else{
		    				params.selectAll = "false";
		    				
		    				var rows = vm.$refs.backupRestoreTaskTable.getData();
		    				if(rows.length != 0){
		    					params.cellCodes = rows.map(function(row){ 
									return row.small_cell_code;
								}).join(',');
		    				}
		    			}

						params.executeMode = vm.ruleForm.executeMode;
						params.cronPeriod = vm.getCronExpression();

						// FTP Server 和执行模式 - 解构赋值
						params.ftpSwitch = vm.ruleForm.ftpSwitch;
						params.ftpUser = vm.ruleForm.ftpUser;
						params.ftpIp = vm.ruleForm.ftpIp;
						params.ftpPort = vm.ruleForm.ftpPort;
						params.ftpPath = vm.ruleForm.ftpPath;
						params.ftpPassword = vm.ruleForm.ftpPassword;
						params.ftpProtocol = vm.ruleForm.ftpProtocol;

						url='${ctx}/task/enb/config/backupRestore/addOrUpdatePeriodBackupTask.action';		    			
					}

					eNBBackupRestoreTasks.slideSubmitLoading = true;

					axios.post(url,stringify(params)).then(function(response){
	    				var data = response.data;

	    				if(data["success"]){
	    					vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success',
	    					})
                            eventBus.$emit('cancel-backupRestore');
	    				}else{
	    					vm.$message.error(data["message"]);
                            eNBBackupRestoreTasks.slideSubmitLoading = false;
	    				}
	    			})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		
		
		// 默认开始时间-时分秒
		defaultStartTimes(){
			var vm = this;

			if(!vm.ruleForm.startTime){
				vm.ruleForm.startTime = formatDate(Date.getNow()).slice(-8)
			}
		},
		// 设置时间-年月日 时分秒
		setTime(){
			var vm = this;

			vm.ruleForm.startTime = formatDate(new Date(gloableTime));
			vm.$refs.ruleForm.validateField('startTime');
		},
		// 关闭新建弹窗
		cancel(){
			var vm = this;

			if(isFormChanged(vm.$refs.ruleForm)){
				vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('cancel-backupRestore')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('cancel-backupRestore')
			}
		},
		//产品类型筛选---------------------------------
		// 高级查询 确定事件
		advanceQuery(type,paramsItem,value){
			var vm = this,
				params = {};

			if(type == 'select'){
				params[paramsItem] = value;
			}else{
				params[paramsItem] = value.join(',');
			}

			if(vm.deviceType == '1'){
				Object.assign(vm.paramsAll, params);
		    }else{
				Object.assign(vm.deviceSelectParams, params);
		    }
		},
        // 清除筛选
        enbClearFilterClick(){
            var vm = this,
                params = {};

            vm.enbAdvancedQueryItemList.map((items)=>{
                if(items.isShow && items.isShow== true ){
                    if(items.type == 'checkbox'){
                        items.checkedItemList = [];
                        items.oldCheckedItemList = [];
                        items.checkAll = false;
                        items.isIndeterminate = false;
                    }else if(items.type == 'select'){
                        items.selectVal = '';
                    }

                    if(items.type == 'checkbox' || items.type == 'select'){
                        params[items.value] = '';
                    }
                }
            })

            if(vm.deviceType == '1'){
                Object.assign(vm.paramsAll, params);
            }else{
                Object.assign(vm.deviceSelectParams, params);
            }

            document.body.click();
        },
	},

	mounted(){
		this.init();
		eventBus.$off('taskSave-ok').$on('taskSave-ok',this.taskSubmit);
		eventBus.$off("show-type").$on("show-type",this.initAddTask);
		eventBus.$off('hander-cancel').$on('hander-cancel',this.cancel);
	}
})
</script>