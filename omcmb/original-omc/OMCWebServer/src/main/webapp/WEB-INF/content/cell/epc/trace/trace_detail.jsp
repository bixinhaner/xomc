<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
	<title>Trace</title>
    <style>
    	.overflow-cls {
			display: inherit;
		}
        #tracePage .tabMainBoxCls{
            display: flex;
            flex-direction: column;
            height: 100%;
        }
        #tracePage .tabMainBoxCls .taskListBoxCls{
            position: relative;
            border-radius: 5px;
            overflow: hidden;
            border-bottom: 1px solid #E9E9E9
        }
        #tracePage .tabMainBoxCls .resultListBoxCls{
            position: relative;
            height: 50%;
            overflow: hidden;
            border-radius: 5px;
        }
        #tracePage .tabMainBoxCls .dividingLineBoxCls{
            height: 1%;
            background-color: #f6f7fb;
        }
        #tracePage .resultListBoxCls .logFileTitle{
            font-size: 14px;
            padding-left: 20px;
            font-weight: 550;
            color:#333333;
        }
        #tracePage .switchBoxCls-ctn {
            display: inline-block;
            position: relative;
            cursor: pointer;
        }
        #tracePage .switchBoxCls-ctn::before {
            content: '';
            position: absolute;
            top: 0;
            right: 0;
            bottom: 0;
            left: 0;
            z-index: 100;
        }
        #tracePage .el-tabs .el-tabs__header{
            border-bottom: 1px solid #DFE2EE;
        }
        #tracePage .el-date-editor .el-range__close-icon {
            line-height:20px;
        }
  	</style>
</head>
<body>
	<!-- enb信令追踪 -->
	<div class="overflow-cls">
		<div id="tracePage" class="container commonWarp leftWarp" style="min-width: 900px;position: relative;">
             <!-- 操作按钮 -->
            <div v-if="isAdmin" class="newIconBoxCls-bt" style="right:17px;top:40px;" @click="openTraceTypeTask" tip="<%=rb.getString("TianJia")%>">
                <span class='el-icon el-icon-circle-add'></span>
            </div>
            <!-- 主页面区域 -- tab页 -->
            <el-tabs v-model="activeName" @tab-click='tabClick' style="height: 100%;">
                <!-- 4G 基站信令追踪 -->
                <el-tab-pane label="<%=rb.getString("eNBXinLingZhuiZong")%>" name="enbTrace">
                    <div class="tabMainBoxCls">
                        <div class="taskListBoxCls" :style="{'height':((enbTraceResultParams.traceId || enbTraceResultParams.traceId === 0) ? '49%' : '100%')}">
                            <!-- 表格组件 -->
                            <el-ctable ref="enbTraceTaskList" id="enbTraceTaskList" :url="enbTraceTaskUrl" :query-params="queryEnbTraceParams" :time="6">
                                <template slot="toolbar">
                                <div class='toolbarHeadBtnBoxCls commonQuery' style="height:40px; padding: 0 0 10px 0;">
                                    <el-query type="normal" @query="query" placeholder="<%=rb.getString("GenZongMingCheng")%>/<%=rb.getString("GenZongCanKaoHao")%>/<%=rb.getString("ShiBieXinXi")%>"></el-query>
                                    <el-date-picker style='margin-left: 10px;' 
                                        key="enbTraceDate"
                                        v-model="enbTraceDateValue"
                                        type="datetimerange"
                                        value-format="yyyy-MM-dd HH:mm:ss"
                                        range-separator="——"  
                                        @change="enbTraceDateChange"
                                        start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
                                        end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
                                    </el-date-picker>
                                    <el-popfilter style="margin: 0 10px;"
                                        label='<%=rb.getString("ZhuangTai")%>'
                                        v-model="queryEnbTraceParams.taskStatus"
                                        type="single"
                                        :list="taskStatusList">
                                    </el-popfilter>
                                </div>
                            </template>					
                                <!-- 列表columns -->
                            <el-table-column prop="op" label=" " width="50" align="center">
                                <template slot-scope="scope">
                                        <div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="optClick(scope.row,event)"></div>
                                </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("GenZongCanKaoHao")%>' width="100" prop="trace_id" sortable></el-table-column>
                                <el-table-column label='<%=rb.getString("GenZongMingCheng")%>' min-width="280" prop="trace_name" sortable></el-table-column>
                                <el-table-column label='<%=rb.getString("JieKouLeiXing")%>' width="200" prop="ne_interface"></el-table-column>
                                <el-table-column label='<%=rb.getString("ShiBieXinXi")%>' width="220" prop="identification_info"></el-table-column>
                            <el-table-column label='<%=rb.getString("ZhuangTai")%>' width="150" prop="task_status">
                                <template slot-scope="scope">
                                    <div v-if="scope.row.task_status == '1'">
                                            <span class="el-icon el-icon-status-waiting1 curStatus"></span>
                                        <span><%=rb.getString("DengDai")%></span>
                                    </div>
                                    <div v-if="scope.row.task_status == '2'">
                                        <span class="el-icon el-icon-status-inProgress curStatus"></span>
                                        <span><%=rb.getString("JinXingZhong")%></span>
                                    </div>
                                    <div v-if="scope.row.task_status == '3'">
                                        <span class="el-icon el-icon-status-terminate curStatus"></span>
                                        <span><%=rb.getString("YiJieShu")%></span>
                                    </div>
                                </template>		        	
                            </el-table-column>
                            <el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="180" prop="start_time" sortable></el-table-column>
                            <el-table-column label='<%=rb.getString("JieShuShiJian")%>' width="180" prop="end_time" sortable></el-table-column>
                            <el-table-column label='<%=rb.getString("ChuangJianZhe")%>' prop="create_user" width="180"></el-table-column>
                                <el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' prop="create_time" width="180" sortable></el-table-column>
                            </el-ctable>
                        </div>
                        <div class="dividingLineBoxCls" v-show="enbTraceResultParams.traceId || enbTraceResultParams.traceId === 0"></div>
                        <div class="resultListBoxCls" v-show="enbTraceResultParams.traceId || enbTraceResultParams.traceId === 0">
                            <!-- 操作按钮 -->
                            <div class="newIconBoxCls-bt" style="right:56px;top:10px;" @click="exportSignaling" tip="<%=rb.getString("DaoChu")%>">
                                <span class='el-icon el-icon-operation-export'></span>
                            </div>
                            <div class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="hideResultClick" tip="<%=rb.getString("GuanBi")%>">
                                <span class='el-icon el-icon-close'></span>
                            </div>
                            <el-ctable ref="enbTraceResultList" id="enbTraceResultTable" :url="enbTraceResultUrl" :query-params="enbTraceResultParams" :time="6">
                                <!-- 头部 -->
                                <template slot="toolbar">
                                    <div class="logFileTitle">
                                        <%=rb.getString("JieGuo")%>
                                        <span style="color:rgba(0,0,0,0.32);">( <%=rb.getString("GenZongCanKaoHao")%>: {{enbTraceResultParams.traceId}} )</span>
                                    </div>
                                </template>
                                <el-table-column prop="interface" label="<%=rb.getString("JieKouLeiXing")%>"></el-table-column>
                                <el-table-column prop="direction" label="<%=rb.getString("FangXiang")%>"></el-table-column>
                                <el-table-column prop="protocol" label="<%=rb.getString("XieYi")%>"></el-table-column>
                                <el-table-column prop="message" label="<%=rb.getString("XiaoXi")%>">
                                    <template slot-scope="scope">
                                        <div>
                                            <a style='color:#1DA3FC;cursor:pointer;' @click='openTrace(scope.row.id)'>{{scope.row.message}}</a>
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="report_time_sec" label="<%=rb.getString("ShiJian")%>">
                                    <template slot-scope="scope">
                                        <div>
                                            <span>{{scope.row.report_time_sec}}.{{scope.row.report_time_ns}}</span>
                                        </div>
                                    </template>
                                </el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </el-tab-pane>
                <!-- 4G 基站UE信令追踪 -->
                <el-tab-pane label="<%=rb.getString("EnbUeXinLingZhuiZong")%>" name="enbUeTrace">
                    <div class="tabMainBoxCls">
                        <div class="taskListBoxCls" :style="{'height':((enbUeTraceResultParams.traceId || enbUeTraceResultParams.traceId === 0) ? '49%' : '100%')}">
                            <!-- 表格组件 -->
                            <el-ctable ref="enbUeTraceTaskList" id="enbUeTraceTaskList" :url="enbUeTraceTaskUrl" :query-params="queryEnbUeTraceParams">
                                <template slot="toolbar">
                                    <div class='toolbarHeadBtnBoxCls commonQuery' style="height:40px; padding: 0 0 10px 0;">
                                        <el-query type="normal" @query="query" placeholder="<%=rb.getString("GenZongCanKaoHao")%>/IMSI"></el-query>
                                        <el-date-picker style='margin-left: 10px;' 
                                            key="enbUeTraceDate"
                                            v-model="enbUeTraceDateValue"
                                            type="datetimerange"
                                            value-format="yyyy-MM-dd HH:mm:ss"
                                            range-separator="——"  
                                            @change="enbUeTraceDateChange"
                                            start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
                                            end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
                                        </el-date-picker>
                                    </div>
                                </template>					
                                    <!-- 列表columns -->
                                <el-table-column prop="op" label=" " width="90" align="center">
                                    <template slot-scope="scope">
                                        <div style="padding-top: 5px;">
                                            <div class="el-icon el-icon-operation-result grayIcon" @click="resultTrace(scope.row,event)" title='<%=rb.getString("PeiZhiJieGuo")%>'></div>
                                            <div v-if="!scope.row.end_time && isAdmin" class="el-icon el-icon-operation-terminate grayIcon" @click="terminateEnbUeTrace(scope.row,event)" style="margin-left: 5px;" title='<%=rb.getString("ZhongZhi")%>'></div>
                                            <div v-if="scope.row.end_time && isAdmin" class="el-icon el-icon-operation-terminate grayIcon disabled" style="margin-left: 5px;" title='<%=rb.getString("ZhongZhi")%>'></div>
                                            <div v-if="scope.row.end_time && isAdmin" class="el-icon el-icon-operation-delete grayIcon" @click="deleteTrace(scope.row,event)" style="margin-left: 5px;" title='<%=rb.getString("ShanChu")%>'></div>
                                            <div v-if="!scope.row.end_time && isAdmin" class="el-icon el-icon-operation-delete grayIcon disabled" style="margin-left: 5px;" title='<%=rb.getString("ShanChu")%>'></div>
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("GenZongCanKaoHao")%>' min-width="100" prop="trace_id" sortable></el-table-column>
                                <el-table-column label='IMSI' min-width="180" prop="imsi"></el-table-column>
                                <el-table-column label='<%=rb.getString("JieKouLeiXing")%>' min-width="140" prop="ne_interface">
                                    <template slot-scope="scope">
                                        <div v-html="interfaceFmt(scope.row.ne_interface)"></div>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("ChiXuShiChang")%>' min-width="110" prop="duration"></el-table-column>
                                <el-table-column label='<%=rb.getString("KaiShiShiJian")%>' min-width="180" prop="start_time" sortable></el-table-column>
                                <el-table-column label='<%=rb.getString("JieShuShiJian")%>' min-width="180" prop="end_time" sortable></el-table-column>
                                <el-table-column label='<%=rb.getString("ChuangJianZhe")%>' prop="create_user" min-width="140"></el-table-column>
                                <el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' prop="create_time" min-width="180" sortable></el-table-column>
                            </el-ctable>
                        </div>
                        <div class="dividingLineBoxCls" v-show="enbUeTraceResultParams.traceId || enbUeTraceResultParams.traceId === 0"></div>
                        <div class="resultListBoxCls" v-show="enbUeTraceResultParams.traceId || enbUeTraceResultParams.traceId === 0">
                            <!-- 操作按钮 -->
                            <div class="newIconBoxCls-bt" style="right:56px;top:10px;" @click="exportSignaling" tip="<%=rb.getString("DaoChu")%>">
                                <span class='el-icon el-icon-operation-export'></span>
                            </div>
                            <div class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="hideResultClick" tip="<%=rb.getString("GuanBi")%>">
                                <span class='el-icon el-icon-close'></span>
                            </div>
                            <el-ctable ref="enbUeTraceResultList" id="enbUeTraceResultList" :url="enbUeTraceResultUrl" :query-params="enbUeTraceResultParams" :time="6">
                                <!-- 头部 -->
                                <template slot="toolbar">
                                    <div class="logFileTitle">
                                        <%=rb.getString("JieGuo")%>
                                        <span style="color:rgba(0,0,0,0.32);">( <%=rb.getString("GenZongCanKaoHao")%>: {{enbUeTraceResultParams.traceId}} )</span>
                                    </div>
                                </template>
                                <el-table-column prop="interface" label="<%=rb.getString("JieKouLeiXing")%>"></el-table-column>
                                <el-table-column prop="direction" label="<%=rb.getString("FangXiang")%>"></el-table-column>
                                <el-table-column prop="protocol" label="<%=rb.getString("XieYi")%>"></el-table-column>
                                <el-table-column prop="message" label="<%=rb.getString("XiaoXi")%>">
                                    <template slot-scope="scope">
                                        <div>
                                            <a style='color:#1DA3FC;cursor:pointer;' @click='openTrace(scope.row.id)'>{{scope.row.message}}</a>
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="report_time_sec" label="<%=rb.getString("ShiJian")%>">
                                    <template slot-scope="scope">
                                        <div>
                                            <span>{{scope.row.report_time_sec}}.{{scope.row.report_time_ns}}</span>
                                        </div>
                                    </template>
                                </el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </el-tab-pane>
                <!-- 2G 基站UE信令追踪 -->
                <el-tab-pane v-if="isSupportGSM" label="<%=rb.getString("GsmUeXinLingZhuiZong")%>" name="gsmUeTrace">
                    <div class="tabMainBoxCls">
                        <div class="taskListBoxCls" :style="{'height':((gsmUeTraceResultParams.traceId || gsmUeTraceResultParams.traceId === 0) ? '49%' : '100%')}">
                            <!-- 表格组件 -->
                            <el-ctable ref="gsmUeTraceTaskList" id="gsmUeTraceTaskList" :url="gsmUeTraceTaskUrl" :query-params="queryGsmUeTraceParams" :time="6">
                                <template slot="toolbar">
                                    <div class='toolbarHeadBtnBoxCls commonQuery' style="height:40px; padding: 0 0 10px 0;">
                                        <el-query type="normal" @query="query" placeholder="<%=rb.getString("GenZongCanKaoHao")%>/IMSI"></el-query>
                                        <el-date-picker style='margin-left: 10px;'
                                            key="gsmUeTraceDate" 
                                            v-model="gsmUeTraceDateValue"
                                            type="datetimerange"
                                            value-format="yyyy-MM-dd HH:mm:ss"
                                            range-separator="——"  
                                            @change="gsmUeTraceDateChange"
                                            start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
                                            end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
                                        </el-date-picker>
                                    </div>
                                </template>					
                                    <!-- 列表columns -->
                                <el-table-column prop="op" label=" " width="90" align="center">
                                    <template slot-scope="scope">
                                        <div style="padding-top: 5px;">
                                            <div class="el-icon el-icon-operation-result grayIcon" @click="resultTrace(scope.row,event)" title='<%=rb.getString("PeiZhiJieGuo")%>'></div>
                                            <div class="el-icon el-icon-operation-info grayIcon" @click="viewGsmUeTraceTask(scope.row,event)" style="cursor: pointer;margin-left: 5px;" title='<%=rb.getString("XinXi")%>'></div>
                                            <div v-if="scope.row.enable !== '1' && isAdmin" class="el-icon el-icon-operation-delete grayIcon" @click="deleteTrace(scope.row,event)" style="margin-left: 5px;" title='<%=rb.getString("ShanChu")%>'></div>
                                            <div v-if="scope.row.enable == '1' && isAdmin" class="el-icon el-icon-operation-delete grayIcon disabled" style="margin-left: 5px;"></div>
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column key="enable" label='Enable' width="80" prop="enable">
                                    <template slot-scope="scope">
                                        <div class="switchBoxCls-ctn" @click="gsmUeTraceEnableChange(scope.row, event)">
                                            <el-switch v-model="scope.row.enable" active-value="1" inactive-value="0" style="zoom: 0.8" ></el-switch>
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("GenZongCanKaoHao")%>' width="100" prop="trace_id" sortable></el-table-column>
                                <el-table-column label='IMSI' min-width="180" prop="imsi"></el-table-column>
                                <el-table-column label='<%=rb.getString("JieKouLeiXing")%>' min-width="120" prop="ne_interface">
                                    <template slot-scope="scope">
                                        <div v-html="gsmInterfaceFmt(scope.row.ne_interface)"></div>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("KaiShiShiJian")%>' min-width="180" prop="start_time" sortable></el-table-column>
                                <el-table-column label='<%=rb.getString("JieShuShiJian")%>' min-width="180" prop="end_time" sortable></el-table-column>
                                <el-table-column label='<%=rb.getString("ChuangJianZhe")%>' prop="create_user" min-width="140"></el-table-column>
                                <el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' prop="create_time" min-width="180" sortable></el-table-column>
                            </el-ctable>
                        </div>
                        <div class="dividingLineBoxCls" v-show="gsmUeTraceResultParams.traceId || gsmUeTraceResultParams.traceId === 0"></div>
                        <div class="resultListBoxCls" v-show="gsmUeTraceResultParams.traceId || gsmUeTraceResultParams.traceId === 0">
                            <!-- 操作按钮 -->
                            <div class="newIconBoxCls-bt" style="right:56px;top:10px;" @click="exportSignaling" tip="<%=rb.getString("DaoChu")%>">
                                <span class='el-icon el-icon-operation-export'></span>
                            </div>
                            <div class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="hideResultClick" tip="<%=rb.getString("GuanBi")%>">
                                <span class='el-icon el-icon-close'></span>
                            </div>
                            <el-ctable ref="gsmUeTraceResultList" id="gsmUeTraceResultList" :url="gsmUeTraceResultUrl" :query-params="gsmUeTraceResultParams" :time="6">
                                <!-- 头部 -->
                                <template slot="toolbar">
                                    <div class="logFileTitle">
                                        <%=rb.getString("JieGuo")%>
                                        <span style="color:rgba(0,0,0,0.32);">( <%=rb.getString("GenZongCanKaoHao")%>: {{gsmUeTraceResultParams.traceId}} )</span>
                                    </div>
                                </template>
                                <el-table-column prop="interface" label="<%=rb.getString("JieKouLeiXing")%>"></el-table-column>
                                <el-table-column prop="direction" label="<%=rb.getString("FangXiang")%>"></el-table-column>
                                <el-table-column prop="protocol" label="<%=rb.getString("XieYi")%>"></el-table-column>
                                <el-table-column prop="message" label="<%=rb.getString("XiaoXi")%>">
                                    <template slot-scope="scope">
                                        <div>
                                            <a style='color:#1DA3FC;cursor:pointer;' @click='openTrace(scope.row.id)'>{{scope.row.message}}</a>
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="report_time_sec" label="<%=rb.getString("ShiJian")%>">
                                    <template slot-scope="scope">
                                        <div>
                                            <span>{{scope.row.report_time_sec}}.{{scope.row.report_time_ns}}</span>
                                        </div>
                                    </template>
                                </el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </el-tab-pane>
            </el-tabs>
            <!-- 右键菜单 -->
            <el-cmenu ref="menu"  @click="menuClick" :data="menus"></el-cmenu>
			<!-- 新建，详情跳转 -->
		    <el-slide ref="addTrace" :title="slideTitle" method='get'
			    :url="slideUrl"  
			    :footer="slideFooter" 
			    :header='slideHeader' 
			    :position="slidePosition"
			    :height="slideHeight" :modal='modal' 
			    :width="slideWidth" 
                :subloading="slideSubmitLoading"
			    @ok='saveTrace'
			    @cancel='cancelAddTrace' 
			    :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'">
				
			</el-slide>
		</div>
	</div>
	<div id="winTest" class="easyui-window" title="<%=rb.getString("XinLingXiaoXi")%>"
	     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:true,width:1050,height:600,resizable:true,inline:false,draggable:true">
	</div>
	<script>
    	//var refreshTaskTimer;
	  	var traceVue = new Vue({
		  	el: '#tracePage',
			data () {
				return {
	          	 	enbTraceTaskUrl: '${ctx}/signaling/querySignalingTraceTaskList.action',
                    enbUeTraceTaskUrl: '${ctx}/enbue/signaling/querySignalingTraceTaskList.action',
                    gsmUeTraceTaskUrl: '${ctx}/bsc/signaling/querySignalingTraceTaskList.action',
                    activeName:'enbTrace',
					queryForm: {
						traceId: '',
						traceName: '',
						identificationInfo: '',
						taskStatus: '',						
	            		timeRange: []
	          		},
                    queryEnbTraceParams: {
			            traceId: '',
						traceName: '',
						identificationInfo: '',
						taskStatus: '',			
			            startTime: '',
			            endTime: '',
			            timeZone: timeZone,
			            searchText: '',
			            likeFileds: 'trace_id,trace_name,identification_info',
			            deviceType: '${ type }',
		            },
                    queryEnbUeTraceParams: {
                        searchText: '',
			            startTime: '',
			            endTime: '',
			            timeZone: timeZone,
			            likeFileds: 'trace_id,imsi',
		            },
                    queryGsmUeTraceParams: {
                        searchText: '',
			            startTime: '',
			            endTime: '',
			            timeZone: timeZone,
			            likeFileds: 'trace_id,imsi',
		            },
		            enbTraceResultUrl: '',
                    enbUeTraceResultUrl: '',
                    gsmUeTraceResultUrl: '',
                    enbTraceResultParams: {
		          		traceId: '',
			            timeZone: timeZone,
		          	},
                    enbUeTraceResultParams: {
                        traceId: '',
                        timeZone: timeZone,
                    },
                    gsmUeTraceResultParams: {
                        traceId: '',
                        timeZone: timeZone,
                    },
	         		menus: [],
	         		rowData: [],
	         		//slide
                    slideUrl: '',
	         		slideTitle:'',
	         		slideHeader:'',
	        	    slideFooter:'',
	        	    slidePosition:'',
	        	    slideHeight:'',
	        	    slideWidth:'',
                    slideSubmitLoading:'',
	        	    operateType:'',
	        	    
	        	    enbTraceDateValue: [],
                    enbUeTraceDateValue: [],
                    gsmUeTraceDateValue: [],
                    taskStatusList:[
                        {value:'',label:'<%=rb.getString("QuanBu")%>'},
                        {value:'1',label:'<%=rb.getString("DengDai")%>'},
                        {value:'2',label:'<%=rb.getString("JinXingZhong")%>'},
                        {value:'3',label:'<%=rb.getString("YiJieShu")%>'}
                    ],
                    isSupportGSM: supportGSM,
		      	}
		    },
            computed: {
				isAdmin(){
					return is_super_user == 'true';
				}
            },
	    	methods: {
                // tab 切换事件
                tabClick(tab){
                    var vm = this;
                },
				optClick(row,ev){
			    	var vm = this, startFlag = false, terminateFlag = false, restartFlag = false, delFlag = false, 
			    		status = row.task_status;
			    	
	    		    vm.rowData = row;
	    		  	//status 1-等待(可开始，可删除)， 2-进行中(可终止，不可删除)，3 已结束 (可重新开始，可删除)
	    		    if(status == '1'){
	    		    	startFlag = true;
	    		    	terminateFlag= false;
	    		    	restartFlag= false;
	    		    	delFlag = true;
		        	}else if(status == '2'){
		        		startFlag = false;
		        		terminateFlag= true;
	    		    	restartFlag= false;
	    		    	delFlag = false;
		        	}else{
		        		startFlag = false;
		        		terminateFlag= false;
	    		    	restartFlag= true;
	    		    	delFlag = true;
		        	}
    		    	vm.menus= [
    			          {label:'<%=rb.getString("JieGuo")%>',cls:"el-icon el-icon-operation-result",code:'result'},
    			          {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start",code:'start',show: startFlag && vm.isAdmin},
    			          {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate",code:'terminate', show: terminateFlag && vm.isAdmin},
    			          {label:'<%=rb.getString("ChongXinKaiShi")%>',cls:"el-icon el-icon-operation-restart",code:'reStart', show: restartFlag && vm.isAdmin},
    			          {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
    			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del',disable:!delFlag && vm.isAdmin}
    			    ]
    		    	this.$nextTick(function(){
    		    		document.body.click();
	    		    	vm.$refs.menu.show(ev);
    		    	});
		        },
	     
				menuClick(ev) {
			        var vm = this,
		              	codes = {
			        		result: vm.resultTrace,
		    	    		start: vm.startTrace,
		    	    		terminate: vm.terminateTrace,
		    	    		reStart: vm.restartTrace,
			    	    	info: vm.infoTrance,
			    	    	del: vm.deleteTrace
		              	};
		
			        if(codes[ev.code]){
			    		codes[ev.code](this.rowData)
			    	}
				},
				
				//结果
				resultTrace(row) {
		            var vm = this;

                    if(vm.activeName == 'enbTrace'){
                        vm.$refs.enbTraceResultList.rows = [];
                        vm.enbTraceResultParams.traceId = row.trace_id;
                        vm.enbTraceResultUrl = '${ctx}/signaling/querySignalingInfoList.action?rd=' + Math.random();
                    }else if(vm.activeName == 'enbUeTrace'){
                        vm.$refs.enbUeTraceResultList.rows = [];
                        vm.rowData = row;
                        vm.enbUeTraceResultParams.traceId = row.trace_id;
                        vm.enbUeTraceResultUrl = '${ctx}/enbue/signaling/querySignalingInfoList.action?rd=' + Math.random();
                    }else{
                        vm.$refs.gsmUeTraceResultList.rows = [];
                        vm.rowData = row;
                        vm.gsmUeTraceResultParams.traceId = row.trace_id;
                        vm.gsmUeTraceResultUrl = '${ctx}/bsc/signaling/querySignalingInfoList.action?rd=' + Math.random();
                    }
				},
                // 关闭结果页面
                hideResultClick(){
                    var vm = this;
                    if(vm.activeName == 'enbTrace'){
                        vm.enbTraceResultUrl = '';
                        vm.enbTraceResultParams.traceId = '';
                    }else if(vm.activeName == 'enbUeTrace'){
                        vm.enbUeTraceResultUrl = '';
                        vm.enbUeTraceResultParams.traceId = '';
                    }else{
                        vm.gsmUeTraceResultUrl = '';
                        vm.gsmUeTraceResultParams.traceId = '';
                    }
                },
				//结果-导出
				exportSignaling(){
					var vm = this,
                        isExistUrl = '',
                        exportUrl = '',
					    params ={
                            timeZone: timeZone,
                            traceId : ''
                        }
                    if(vm.activeName == 'enbTrace'){
                        params.traceId = vm.enbTraceResultParams.traceId;
                        isExistUrl = '${ctx}/signaling/createSignalingPcapFile.action';
                        exportUrl = '${ctx}/signaling/exportSignalingPcapFile.action';
                    }else if(vm.activeName == 'enbUeTrace'){
                        params.traceId = vm.enbUeTraceResultParams.traceId;
                        isExistUrl = '${ctx}/enbue/signaling/createSignalingPcapFile.action';
                        exportUrl = '${ctx}/enbue/signaling/exportSignalingPcapFile.action';
                    }else{
                        params.traceId = vm.gsmUeTraceResultParams.traceId;
                        isExistUrl = '${ctx}/bsc/signaling/createSignalingPcapFile.action';
                        exportUrl = '${ctx}/bsc/signaling/exportSignalingPcapFile.action';
                    }
					axios.post(isExistUrl,stringify(params)).then(function(response){
						var data = response.data;
			            if(data["success"]){
			            	exportByForm(exportUrl,params);
			            }else{
			              	vm.$message.error(data["message"])
			            }
			    	})
				},
				//message 点击 查看详细信令   二进制  树相互转换
				openTrace(id){
					var vm = this,
                        isExistUrl = '',
                        infoUrl = '',
                        params = {
                            id: id
                        };

                    if(vm.activeName == 'enbTrace'){
                        isExistUrl = '${ctx}/signaling/isExistSignalingRelateFile.action';
                        infoUrl = '${ctx}/signaling/toSignalingDetailInfo.action?id='+id;
                    }else if(vm.activeName == 'enbUeTrace'){
                        isExistUrl = '${ctx}/enbue/signaling/isExistSignalingRelateFile.action';
                        infoUrl = '${ctx}/enbue/signaling/toSignalingDetailInfo.action?id='+id;
                    }else{
                        isExistUrl = '${ctx}/bsc/signaling/isExistSignalingRelateFile.action';
                        infoUrl = '${ctx}/bsc/signaling/toSignalingDetailInfo.action?id='+id;
                    }
					axios.post(isExistUrl,stringify(params)).then(function(response){
						var data = response.data;
			            if(data["success"]){
			            	$("#winTest").window("open").window('center'); 
			            	$("#winTest").window("refresh", infoUrl);
			            }else{
			              	vm.$message.error('<%=rb.getString("XinLinXiaoXiBuCunZai")%>')
			            }
			    	})
				},
				//开始
				startTrace(row) {
          			var vm = this;

			        axios.post('${ctx}/signaling/startSignalingTraceTask.action',stringify({
			        	traceId : row.trace_id
			        })).then(function(response){
						var data = response.data;
			            if(data["success"]){
			            	vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
			              	vm.$refs.enbTraceTaskList.refresh()
			            }else{
			              	vm.$message.error(data["message"])
			            }
			    	})
        		},
        		//终止
        		terminateTrace(row) {
		        	var vm = this;
			    	    	
		          	axios.post('${ctx}/signaling/terminateSignalingTraceTask.action',stringify({
		          		traceId : row.trace_id
		          	})).then(function(response){
		            	var data = response.data;
			            if(data["success"]){
			            	vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
			                vm.$refs.enbTraceTaskList.refresh()
			            }else{
			                vm.$message.error(data["message"])
			            }
		          	})
		        },
		      	//重新开始
			    restartTrace(row){
					var vm = this;
					axios.post('${ctx}/signaling/restartSignalingTraceTask.action',stringify({
						traceId :  row.trace_id
		            })).then(function(response){
		                var data = response.data;
		                if(data["success"]){
		                	vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
		                    vm.$refs.enbTraceTaskList.refresh()
		                }else{
		                    vm.$message.error(data["message"])
		                }
		            })
				},
				//新建任务
				openTraceTypeTask(){
          			var vm = this;
          			
          			vm.operateType = 'add';
          			vm.slideHeader = true;
			    	vm.slideFooter = true;
			    	vm.slidePosition = 'top';
			    	vm.slideHeight = '100%';
			    	vm.slideWidth = '100%';
                    vm.slideSubmitLoading = false;
			    	
                    if(vm.activeName == 'enbTrace'){
                        vm.slideTitle = '<%=rb.getString("eNBGenZongRenWuChuangJian")%>';
                        vm.slideUrl = '${ctx}/signaling/toAddENBSignalingTracePage.action?type=&timeZone=' + timeZone;
                    }else if(vm.activeName == 'enbUeTrace'){
                        vm.slideTitle = '<%=rb.getString("XinJianEnbUeXinLingZhuiZong")%>';
                        vm.slideUrl = '${ctx}/enbue/signaling/toAddENBUESignalingTracePage.action?timeZone=' + timeZone;
                    }else{
                        vm.slideTitle = '<%=rb.getString("XinJianGsmUeXinLingZhuiZong")%>';
                        vm.slideUrl = '${ctx}/bsc/signaling/toAddGSMSignalingTracePage.action?timeZone=' + timeZone;
                    }
					vm.$refs.addTrace.showSlide(function(){
						vm.modal = false;
						eventBus.$emit('init-config','add', '');
				    });
	      		},
                // 2G UE信令追踪 信息
                viewGsmUeTraceTask(row){
                    var vm = this;
                    vm.operateType = 'view';
                    vm.slideHeader = true;
                    vm.slideFooter = false;
                    vm.slideUrl = '${ctx}/bsc/signaling/toAddGSMSignalingTracePage.action?timeZone=' + timeZone;
                    vm.slidePosition = 'top';
                    vm.slideHeight = '100%';
                    vm.slideWidth = '100%';
                    vm.slideTitle = '<%=rb.getString("XinXi")%>';
                    vm.$refs.addTrace.showSlide(function(){
                        vm.modal = false;
                        eventBus.$emit('init-config','view', row);
                    });
                },
                // 4G UE信令追踪 任务终止
                terminateEnbUeTrace(row){
                    var vm = this;
                    axios.post('${ctx}/enbue/signaling/terminateSignalingTraceTask.action',stringify({
                        traceId : row.trace_id
                    })).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type: 'success'
                            });
                            vm.$refs.enbUeTraceTaskList.refresh()
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                },
		        saveTrace() {
	      			var vm = this;
	      			eventBus.$emit('hander-ok')
		        },
		      	//关闭新建，详情
			    cancelAddTrace(){
		        	var vm = this;
		        	if(vm.operateType == 'view'){
		        		vm.$refs.addTrace.hide();
			    	}else{
			    		eventBus.$emit('close-slide')
			    	}
        		},
		      	//信息
				infoTrance(row){
			    	var vm = this;
			    	
			    	vm.operateType = 'view';
			    	vm.slideHeader = true;
			    	vm.slideFooter = false;
			    	vm.slideUrl = '${ctx}/signaling/querySignalingTraceProperties.action?traceId=' + row.trace_id + '&timeZone=' + timeZone;
			    	vm.slidePosition = 'top';
			    	vm.slideHeight = '100%';
			    	vm.slideWidth = '100%';
                    vm.slideSubmitLoading = false;
			    	vm.slideTitle = '<%=rb.getString("XinXi")%>';
					vm.$refs.addTrace.showSlide(function(){
						vm.modal = false;
						eventBus.$emit('init-config','readonly', row);
				    });
			    },
				//删除
		        deleteTrace(row) {
		        	var vm = this,
                        urls = '';
                    if(vm.activeName == 'enbTrace'){
                        urls = '${ctx}/signaling/delSignalingTraceTask.action';
                    }else if(vm.activeName == 'enbUeTrace'){
                        urls = '${ctx}/enbue/signaling/delSignalingTraceTask.action';
                    }else{
                        urls = '${ctx}/bsc/signaling/delSignalingTraceTask.action';
                    }
		            vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
			            customClass:'warningConfirm',
			            confirmButtonText:'<%=rb.getString("QueDing")%>',
			            cancelButtonText:'<%=rb.getString("QuXiao")%>',
			            type:'warning',
			            closeOnClickModal:false
		          	}).then(() => {
			            axios.post(urls,stringify({
			            	traceId: row.trace_id
			            })).then(function(response){
							var data = response.data;
				            if(data["success"]){
				                vm.$message({
				                	type:'success',
				                    message:'<%=rb.getString("ChengGong")%>'
				                })
                                if(vm.activeName == 'enbTrace'){
                                    vm.$refs.enbTraceTaskList.refresh();
                                    if(vm.enbTraceResultParams.traceId == row.trace_id){
                                        vm.enbTraceResultParams.traceId = '';
                                    }
                                }else if(vm.activeName == 'enbUeTrace'){
                                    vm.$refs.enbUeTraceTaskList.refresh();
                                    if(vm.enbUeTraceResultParams.traceId == row.trace_id){
                                        vm.enbUeTraceResultParams.traceId = '';
                                    }
                                }else{
                                    vm.$refs.gsmUeTraceTaskList.refresh();
                                    if(vm.gsmUeTraceResultParams.traceId == row.trace_id){
                                        vm.gsmUeTraceResultParams.traceId = '';
                                    }
                                }
				            }else{
				             	vm.$message.error(data["message"])
				            }
			            }).catch(function(error){})
		          	}).catch(function(error){})
		        },
                // 模糊搜索
		        query(text) {
		          	var vm = this;
                    if(vm.activeName == 'enbTrace'){
                        vm.queryEnbTraceParams.searchText = text;
                    }else if(vm.activeName == 'enbUeTrace'){
                        vm.queryEnbUeTraceParams.searchText = text;
                    }else{
                        vm.queryGsmUeTraceParams.searchText = text;
                    }
		        },
		        // 操作菜单关闭
				hideMenus() {
                    var vm = this;
			        vm.$refs.menu.hide()
			    },
			    hideTrace(){
					var vm = this;
			    	vm.$refs.addTrace.hide();
                    if(vm.activeName == 'enbTrace'){
                        vm.$refs.enbTraceTaskList.refresh()
                    }else if(vm.activeName == 'enbUeTrace'){
                        vm.$refs.enbUeTraceTaskList.refresh()
                    }else{
                        vm.$refs.gsmUeTraceTaskList.refresh()
                    }
			    },
			    // 4G 基站信令追踪 时间筛选
			    enbTraceDateChange(val) {
					var vm = this;
                    vm.enbTraceDateValue = val;
                    if(val != null){
                        vm.queryEnbTraceParams.startTime = vm.enbTraceDateValue[0];
                        vm.queryEnbTraceParams.endTime = vm.enbTraceDateValue[1];
                    }else{
                        vm.queryEnbTraceParams.startTime = '';
                        vm.queryEnbTraceParams.endTime = '';
                    }
				},
                // 4G UE信令追踪 时间筛选
                enbUeTraceDateChange(val) {
					var vm = this;
                    vm.enbUeTraceDateValue = val;
                    if(val != null){
                        vm.queryEnbUeTraceParams.startTime = vm.enbUeTraceDateValue[0];
                        vm.queryEnbUeTraceParams.endTime = vm.enbUeTraceDateValue[1];
                    }else{
                        vm.queryEnbUeTraceParams.startTime = '';
                        vm.queryEnbUeTraceParams.endTime = '';
                    }
				},
                // 2G UE信令追踪 时间筛选
                gsmUeTraceDateChange(val){
                    var vm = this;
                    vm.gsmUeTraceDateValue = val;
                    if(val != null){
                        vm.queryGsmUeTraceParams.startTime = vm.gsmUeTraceDateValue[0];
                        vm.queryGsmUeTraceParams.endTime = vm.gsmUeTraceDateValue[1];
                    }else{
                        vm.queryGsmUeTraceParams.startTime = '';
                        vm.queryGsmUeTraceParams.endTime = '';
                    }
                },
                // 2G UE 信令追踪开关改变事件
                gsmUeTraceEnableChange(row){
                    var vm = this,
                        urls = '',
                        confirmStr = '',
                        params = {
                            traceId: row.trace_id,
                            enable:row.enable == '1'? '0' : '1',
                            imsi: row.imsi,
                            NEInterface: row.ne_interface,
                        };
                    if(row.enable == '1'){
                        urls = '${ctx}/bsc/signaling/disableSignalingTraceTask.action';
                        confirmStr = '<%=rb.getString("QueRenGuanBi")%>'
                    }else{
                        urls = '${ctx}/bsc/signaling/enableSignalingTraceTask.action';
                        confirmStr = '<%=rb.getString("QueRenKaiQi")%>'
                    }
                    vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
			            customClass:'warningConfirm',
			            confirmButtonText:'<%=rb.getString("QueDing")%>',
			            cancelButtonText:'<%=rb.getString("QuXiao")%>',
			            type:'warning',
			            closeOnClickModal:false
		          	}).then(() => {
			            axios.post(urls,stringify(params)).then(function(response){
                            var data = response.data;
                            var message = '<%=rb.getString("ChengGong")%>';
                            if(data["success"]){
                                vm.$message({
                                    message:message,
                                    type:'success',
                                })
                                vm.$refs.gsmUeTraceTaskList.refresh();
                            }else{
                                vm.$message.error(data["message"])
                            }
                        })
		          	}).catch(function(error){})
                    
                    event.stopPropagation();
                },
                interfaceFmt(val){
                    var str = '',
                        codes = {
                            '0': 'S1',
                            '1': 'Uu',
                            '2': 'Xn',
                            '3': 'X2',
                            '4': 'F1-C',
                        },
                        list =[];
                    if(val){
                        list = val.split(',');
                        list.map((item)=>{
                            str += codes[item] + ',';
                        })
                        str = str.substring(0,str.length-1);
                    }
                    return str;
                },
                gsmInterfaceFmt(val){
                    var str = '',
                        codes = {
                            '1': 'A-Interface',
                            '2': 'Abits-Interface',
                        },
                        list =[];
                    if(val){
                        list = val.split(',');
                        list.map((item)=>{
                            str += codes[item] + ',';
                        })
                        str = str.substring(0,str.length-1);
                    }
                    return str;
                },
                
	    	},
	    	watch:{},
			mounted() {
		        var vm = this;
		        eventBus.$off("hander-cancel").$on('hander-cancel',this.hideTrace);
			}
		})
 	</script>
</body>
</html>